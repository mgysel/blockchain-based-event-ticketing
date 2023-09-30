package integration

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "net/http/pprof"

	"github.com/stretchr/testify/require"
	accessContract "go.dedis.ch/dela/contracts/access"
	"go.dedis.ch/dela/core/txn"
	"go.dedis.ch/dela/core/txn/signed"
	"go.dedis.ch/dela/crypto/bls"
	"go.dedis.ch/dela/crypto/loader"
	"go.dedis.ch/dela/dkg"
	"go.dedis.ch/dela/dkg/pedersen"
	"go.dedis.ch/dela/dkg/pedersen/types"

	"go.dedis.ch/dela/mino"

	"go.dedis.ch/dela/mino/minoch"
	"go.dedis.ch/dela/mino/minogrpc"
	"go.dedis.ch/dela/mino/router/tree"

	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/kyber/v3/sign/bdn"
	"go.dedis.ch/kyber/v3/suites"
	"go.dedis.ch/kyber/v3/xof/keccak"
)

func init() {
	rand.Seed(0)
}

func Test_F3B_Original(t *testing.T) {
	batchSizes := []int{50}
	numDKGs := []int{64}
	// numDKGs := []int{3}
	withGrpc := false

	for _, batchSize := range batchSizes {
		for _, numDKG := range numDKGs {
			t.Run(fmt.Sprintf("batch size %d num dkg %d", batchSize, numDKG),
				f3bScenario(batchSize, numDKG, 3, withGrpc))
		}
	}
}

func f3bScenario(batchSize, numDKG, numNodes int, withGrpc bool) func(t *testing.T) {
	return func(t *testing.T) {

		require.Greater(t, numDKG, 0)
		require.Greater(t, numNodes, 0)
		require.GreaterOrEqual(t, numDKG, numNodes)

		to := time.Second * 10 // transaction inclusion timeout

		// set up the dkg
		minosBuilder := getMinoch
		if withGrpc {
			minosBuilder = getMinogRPCs
		}

		minos := minosBuilder(t, numDKG)
		dkgs := make([]dkg.DKG, numDKG)
		addrs := make([]mino.Address, numDKG)

		// initializing the addresses
		for i, mino := range minos {
			addrs[i] = mino.GetAddress()
		}

		pubkeys := make([]kyber.Point, len(minos))
		bdnPubkeys := make([]kyber.Point, len(minos))
		bdnSuite := pairing.NewSuiteBn256()

		for i, mino := range minos {
			dkg, pubkey, _ := pedersen.NewPedersen(mino)
			dkgs[i] = dkg
			pubkeys[i] = pubkey
			_, bdnPk := bdn.NewKeyPair(bdnSuite, bdnSuite.RandomStream())
			bdnPubkeys = append(bdnPubkeys, bdnPk)
		}

		actors := make([]dkg.Actor, numDKG)
		for i := 0; i < numDKG; i++ {
			actor, err := dkgs[i].Listen()
			require.NoError(t, err)
			actors[i] = actor
		}

		fakeAuthority := NewAuthority(addrs, pubkeys)
		fakeAuthorityBls := NewAuthorityBls(addrs, bdnPubkeys)
		start := time.Now()
		_, err := actors[0].Setup(fakeAuthority, fakeAuthorityBls, numDKG)
		require.NoError(t, err)

		setupTime := time.Since(start)
		fmt.Println("setup done in ", setupTime)
		t.Logf("setup done in %s", setupTime)

		// setting up the blockchain

		dir, err := os.MkdirTemp("", "dela-integration-test")
		require.NoError(t, err)

		t.Logf("using temps dir %s", dir)

		defer os.RemoveAll(dir)

		nodes := make([]dela, numNodes)

		for i := range nodes {
			nodes[i] = newDelaNode(t, filepath.Join(dir, fmt.Sprintf("node%d", i)), 0)
		}

		nodes[0].Setup(nodes[1:]...)

		l := loader.NewFileLoader(filepath.Join(dir, "private.key"))

		// creating a new client/signer
		signerdata, err := l.LoadOrCreate(newKeyGenerator())
		require.NoError(t, err)

		signer, err := bls.NewSignerFromBytes(signerdata)
		require.NoError(t, err)

		pubKey := signer.GetPublicKey()
		cred := accessContract.NewCreds(aKey[:])

		for _, node := range nodes {
			node.GetAccessService().Grant(node.(cosiDelaNode).GetAccessStore(), cred, pubKey)
		}

		manager := signed.NewManager(signer, &txClient{})

		pubKeyBuf, err := signer.GetPublicKey().MarshalBinary()
		require.NoError(t, err)

		// sending the grant transaction to the blockchain
		args := []txn.Arg{
			{Key: "go.dedis.ch/dela.ContractArg", Value: []byte("go.dedis.ch/dela.Access")},
			{Key: "access:grant_id", Value: []byte(hex.EncodeToString(valueAccessKey[:]))},
			{Key: "access:grant_contract", Value: []byte("go.dedis.ch/dela.Value")},
			{Key: "access:grant_command", Value: []byte("all")},
			{Key: "access:identity", Value: []byte(base64.StdEncoding.EncodeToString(pubKeyBuf))},
			{Key: "access:command", Value: []byte("GRANT")},
		}

		// waiting for the confirmation of the transaction
		err = addAndWait(t, to, manager, nodes[0].(cosiDelaNode), args...)
		require.NoError(t, err)

		// creating GBar. we need a generator in order to follow the encryption and
		// decryption protocol of https://arxiv.org/pdf/2205.08529.pdf / we take an
		// agreed data among the participants and embed it as a point. the result is
		// the generator that we are seeking
		var suite = suites.MustFind("Ed25519")
		agreedData := make([]byte, 32)
		_, err = rand.Read(agreedData)
		require.NoError(t, err)
		gBar := suite.Point().Embed(agreedData, keccak.New(agreedData))

		// creating the symmetric keys in batch. we process the transactions in
		// batch to increase the throughput for more information refer to
		// https://arxiv.org/pdf/2205.08529.pdf / page 6 / step 1 (write
		// transaction)

		// the write transaction arguments
		argSlice := make([][]txn.Arg, batchSize)

		var ciphertexts []types.Ciphertext

		// generate random messages to be encrypted
		keys := make([][29]byte, batchSize)
		for i := range keys {
			_, err = rand.Read(keys[i][:])
			require.NoError(t, err)
		}

		start = time.Now()

		// Create a Write instance
		for i := 0; i < batchSize; i++ {
			// Encrypting the symmetric key
			ciphertext, remainder, err := actors[0].VerifiableEncrypt(keys[i][:], gBar)
			require.NoError(t, err)
			require.Len(t, remainder, 0)

			ciphertexts = append(ciphertexts, ciphertext)

			// converting the kyber.Point or kyber.Scalar to bytes
			Cbytes, err := ciphertext.C.MarshalBinary()
			require.NoError(t, err)
			Ubytes, err := ciphertext.K.MarshalBinary()
			require.NoError(t, err)
			Ubarbytes, err := ciphertext.UBar.MarshalBinary()
			require.NoError(t, err)
			Ebytes, err := ciphertext.E.MarshalBinary()
			require.NoError(t, err)
			Fbytes, err := ciphertext.F.MarshalBinary()
			require.NoError(t, err)

			// put all the data together
			Ck := append(Cbytes[:], Ubytes[:]...)
			Ck = append(Ck, Ubarbytes[:]...)
			Ck = append(Ck, Ebytes[:]...)
			Ck = append(Ck, Fbytes[:]...)

			// creating the transaction and write the data
			// NOTE: This is writing the encrypted symmetric key
			argSlice[i] = []txn.Arg{
				{Key: "go.dedis.ch/dela.ContractArg", Value: []byte("go.dedis.ch/dela.Value")},
				{Key: "value:key", Value: []byte("key")},
				{Key: "value:value", Value: Ck},
				{Key: "value:command", Value: []byte("WRITE")},
			}

			// we read the recorded data on the blockchain and make sure that
			// the data was submitted correctly
			err = addAndWait(t, to, manager, nodes[0].(cosiDelaNode), argSlice[i]...)
			require.NoError(t, err)

			// Make sure value tx correct
			proof, err := nodes[0].GetOrdering().GetProof([]byte("key"))
			require.NoError(t, err)
			require.Equal(t, Ck, proof.GetValue())
		}

		submitTime := time.Since(start)
		t.Logf("submit batch: %s", submitTime)

		start = time.Now()

		// decrypting the symmetric key in batch
		fmt.Println("Decrypting symmetric key")
		decrypted, err := actors[0].VerifiableDecrypt(ciphertexts)
		require.NoError(t, err)

		decryptTime := time.Since(start)
		t.Logf("decrypt batch: %s", decryptTime)

		// make sure that the decryption was correct
		fmt.Println("Check decryption")
		for i := 0; i < batchSize; i++ {
			require.Equal(t, keys[i][:], decrypted[i])
		}

		fmt.Println("Setup,\tSubmit,\tDecrypt")
		fmt.Printf("%d,\t%d,\t%d\n", setupTime.Milliseconds(), submitTime.Milliseconds(), decryptTime.Milliseconds())
	}
}

func getMinoch(t *testing.T, n int) []mino.Mino {
	res := make([]mino.Mino, n)

	minoManager := minoch.NewManager()

	for i := range res {
		minoch := minoch.MustCreate(minoManager, fmt.Sprintf("addr %d", i))
		res[i] = minoch
	}

	return res
}

func getMinogRPCs(t *testing.T, n int) []mino.Mino {
	res := make([]mino.Mino, n)

	addr := minogrpc.ParseAddress("127.0.0.1", uint16(0))
	router := tree.NewRouter(minogrpc.NewAddressFactory(), tree.WithHeight(1))

	for i := range res {
		grpc, err := minogrpc.NewMinogrpc(addr, nil, router, minogrpc.DisableTLS())
		require.NoError(t, err)

		res[i] = grpc
	}

	return res
}
