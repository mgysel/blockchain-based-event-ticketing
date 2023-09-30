package controller

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.dedis.ch/dela/cli/node"
	"go.dedis.ch/dela/core/ordering/cosipbft/authority"
	"go.dedis.ch/dela/crypto"
	"go.dedis.ch/dela/crypto/bls"
	"go.dedis.ch/dela/crypto/ed25519"
	"go.dedis.ch/dela/dkg"
	mTypes "go.dedis.ch/dela/dkg/pedersen/types"
	"go.dedis.ch/dela/mino"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/pairing"
	"go.dedis.ch/kyber/v3/suites"
	"golang.org/x/xerrors"
)

// suite is the Kyber suite for Pedersen.
var suite = suites.MustFind("Ed25519")

// var blsSuite = suites.MustFind("bls")
var bdnSuite = pairing.NewSuiteBn256()

const separator = ":"
const authconfig = "dkgauthority"
const resolveActorFailed = "failed to resolve actor, did you call listen?: %v"

type setupAction struct{}

func (a setupAction) Execute(ctx node.Context) error {
	var actor dkg.Actor

	err := ctx.Injector.Resolve(&actor)
	if err != nil {
		return xerrors.Errorf(resolveActorFailed, err)
	}

	co, coBdn, err := getCollectiveAuth(ctx)
	if err != nil {
		return xerrors.Errorf("failed to get collective authority: %v", err)
	}

	t := ctx.Flags.Int("threshold")

	pubkey, err := actor.Setup(co, coBdn, t)
	if err != nil {
		return xerrors.Errorf("failed to setup: %v", err)
	}

	fmt.Fprintf(ctx.Out, "✅ Setup done.\n🔑 Pubkey: %s", pubkey.String())

	return nil
}

func getCollectiveAuth(ctx node.Context) (crypto.CollectiveAuthority, crypto.CollectiveAuthority, error) {
	fmt.Println("*** Inside getCollectiveAuth")
	authorities := ctx.Flags.StringSlice("authority")
	fmt.Println("Authorities: ", authorities)

	addrs := make([]mino.Address, len(authorities))

	pubkeys := make([]crypto.PublicKey, len(authorities))

	bdnPubkeys := make([]crypto.PublicKey, len(authorities))

	for i, auth := range authorities {
		addr, pk, bdnPk, err := decodeAuthority(ctx, auth)
		if err != nil {
			return nil, nil, xerrors.Errorf("failed to decode authority: %v", err)
		}

		addrs[i] = addr
		pubkeys[i] = ed25519.NewPublicKeyFromPoint(pk)
		bdnPubkeys[i] = bls.NewPublicKeyFromPoint(bdnPk)

	}

	co := authority.New(addrs, pubkeys)
	coBdn := authority.New(addrs, bdnPubkeys)

	return co, coBdn, nil
}

// func getCollectiveAuth(ctx node.Context) (crypto.CollectiveAuthority, error) {
// 	fmt.Println("*** Inside getCollectiveAuth")
// 	authorities := ctx.Flags.StringSlice("authority")
// 	fmt.Println("Authorities: ", authorities)

// 	addrs := make([]mino.Address, len(authorities))

// 	pubkeys := make([]crypto.PublicKey, len(authorities))

// 	bdnPubkeys := make([]crypto.PublicKey, len(authorities))

// 	for i, auth := range authorities {
// 		addr, pk, bdnPk, err := decodeAuthority(ctx, auth)
// 		if err != nil {
// 			return nil, xerrors.Errorf("failed to decode authority: %v", err)
// 		}

// 		addrs[i] = addr
// 		pubkeys[i] = ed25519.NewPublicKeyFromPoint(pk)
// 		bdnPubkeys[i] = bls.NewPublicKeyFromPoint(bdnPk)

// 	}

// 	co := authority.New(addrs, pubkeys)
// 	coBdn := authority.New(addrs, bdnPubkeys)

// 	return co, nil
// }

type listenAction struct {
	pubkey    kyber.Point
	bdnPubkey kyber.Point
}

func (a listenAction) Execute(ctx node.Context) error {
	var dkg dkg.DKG

	err := ctx.Injector.Resolve(&dkg)
	if err != nil {
		return xerrors.Errorf("failed to resolve dkg: %v", err)
	}

	actor, err := dkg.Listen()
	if err != nil {
		return xerrors.Errorf("failed to listen: %v", err)
	}

	ctx.Injector.Inject(actor)

	fmt.Fprintf(ctx.Out, "✅  Listen done, actor is created.")

	str, err := encodeAuthority(ctx, a.pubkey, a.bdnPubkey)
	if err != nil {
		return xerrors.Errorf("failed to encode authority: %v", err)
	}

	path := filepath.Join(ctx.Flags.Path("config"), authconfig)

	err = os.WriteFile(path, []byte(str), 0755)
	if err != nil {
		return xerrors.Errorf("failed to write authority configuration: %v", err)
	}

	fmt.Fprintf(ctx.Out, "📜 Config file written in %s", path)

	return nil
}

func encodeAuthority(ctx node.Context, pk kyber.Point, bdnPk kyber.Point) (string, error) {
	var m mino.Mino
	err := ctx.Injector.Resolve(&m)
	if err != nil {
		return "", xerrors.Errorf("failed to resolve mino: %v", err)
	}

	addr, err := m.GetAddress().MarshalText()
	if err != nil {
		return "", xerrors.Errorf("failed to marshal address: %v", err)
	}

	pkbuf, err := pk.MarshalBinary()
	if err != nil {
		return "", xerrors.Errorf("failed to marshall pubkey: %v", err)
	}

	bdnPkbuf, err := bdnPk.MarshalBinary()
	if err != nil {
		return "", xerrors.Errorf("failed to marshall bdn pubkey: %v", err)
	}

	id := base64.StdEncoding.EncodeToString(addr) + separator +
		base64.StdEncoding.EncodeToString(pkbuf) + separator +
		base64.StdEncoding.EncodeToString(bdnPkbuf)

	return id, nil
}

func decodeAuthority(ctx node.Context, str string) (mino.Address, kyber.Point, kyber.Point, error) {
	fmt.Println("*** Inside decodeAuthority")
	fmt.Println("str: ", str)

	parts := strings.Split(str, separator)
	if len(parts) != 3 {
		return nil, nil, nil, xerrors.New("invalid identity base64 string")
	}
	fmt.Println("parts: ", parts)

	// 1. Deserialize the address.
	var m mino.Mino
	err := ctx.Injector.Resolve(&m)
	if err != nil {
		return nil, nil, nil, xerrors.Errorf("injector: %v", err)
	}

	addrBuf, err := base64.StdEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, nil, nil, xerrors.Errorf("base64 address: %v", err)
	}

	addr := m.GetAddressFactory().FromText(addrBuf)
	fmt.Println("addr: ", addr)

	// 2. Deserialize the public key.
	pubkeyBuf, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, nil, nil, xerrors.Errorf("base64 public key: %v", err)
	}

	pubkey := suite.Point()

	err = pubkey.UnmarshalBinary(pubkeyBuf)
	if err != nil {
		return nil, nil, nil, xerrors.Errorf("failed to decode pubkey: %v", err)
	}

	// 3. Deserialize the bdn public key.
	bdnPubkeyBuf, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, nil, nil, xerrors.Errorf("base64 bdn public key: %v", err)
	}

	bdnPubkey := bdnSuite.Point()

	err = bdnPubkey.UnmarshalBinary(bdnPubkeyBuf)
	if err != nil {
		return nil, nil, nil, xerrors.Errorf("failed to decode bdn pubkey: %v", err)
	}
	fmt.Println("bdn pubkey: ", pubkey)

	return addr, pubkey, bdnPubkey, nil
}

type issueMasterCredentialAction struct{}

func credentialToString(credential []byte) string {
	credentialString := string(credential)

	return credentialString
}

func stringToCredential(credential string) []byte {
	credentialBytes := []byte(credential)

	return credentialBytes
}

func signaturesToString(signatures [][]byte) string {
	signatureString := ""
	for i := 0; i < len(signatures); i++ {
		if i == 0 {
			signatureString = string(signatures[i])
		} else {
			signatureString = signatureString + ":" + string(signatures[i])
		}
	}

	return signatureString
}

func stringToSignature(signatures string) [][]byte {
	split := strings.Split(signatures, ":")
	signatureBytes := make([][]byte, len(split))
	for i := 0; i < len(split); i++ {
		signatureBytes[i] = []byte(split[i])
	}

	return signatureBytes
}

func (a issueMasterCredentialAction) Execute(ctx node.Context) error {
	fmt.Println("*** action - Inside issueMasterCredential Action")
	var actor dkg.Actor

	err := ctx.Injector.Resolve(&actor)
	if err != nil {
		return xerrors.Errorf(resolveActorFailed, err)
	}

	id_hash := ctx.Flags.String("idhash")
	fmt.Println("ID Hash: ", id_hash)
	fmt.Println("Actor: ", actor)

	masterCred, masterSignatures, err := actor.IssueMasterCredential(id_hash)
	if err != nil {
		return xerrors.Errorf("failed to issue master credential: %v", err)
	}

	out := base64.StdEncoding.EncodeToString(masterCred)
	for i := 0; i < len(masterSignatures); i++ {
		out = out + separator + base64.StdEncoding.EncodeToString(masterSignatures[i])
	}

	fmt.Println("OUT: ", out)

	fmt.Fprint(ctx.Out, out)

	return nil
}

type issueEventCredentialAction struct{}

func (a issueEventCredentialAction) Execute(ctx node.Context) error {
	fmt.Println("*** action - Inside issueEventCredential Action")
	var actor dkg.Actor

	err := ctx.Injector.Resolve(&actor)
	if err != nil {
		return xerrors.Errorf(resolveActorFailed, err)
	}

	fmt.Println("flags: ", ctx.Flags)

	idHash := ctx.Flags.String("idhash")
	eventName := ctx.Flags.String("eventName")
	masterCredentialString := ctx.Flags.String("masterCredential")
	masterCredential, err := base64.StdEncoding.DecodeString(masterCredentialString)
	if err != nil {
		return xerrors.Errorf("failed to decode master credential: %v", err)
	}
	masterSignaturesString := ctx.Flags.String("masterSignatures")
	masterSignaturesSplit := strings.Split(masterSignaturesString, ":")
	masterSignatures := [][]byte{}
	for i := 0; i < len(masterSignaturesSplit); i++ {
		thisMasterSignature, err := base64.StdEncoding.DecodeString(masterSignaturesSplit[i])
		if err != nil {
			return xerrors.Errorf("failed to decode master signature: %v", err)
		}
		masterSignatures = append(masterSignatures, thisMasterSignature)
	}
	fmt.Println("Master Credential: ", masterCredential)
	fmt.Println("Master Signatures: ", masterSignatures)
	fmt.Println("Actor: ", actor)

	eventCred, eventSignatures, err := actor.IssueEventCredential(idHash, eventName, masterCredential, masterSignatures)
	if err != nil {
		return xerrors.Errorf("failed to issue event credential: %v", err)
	}

	out := base64.StdEncoding.EncodeToString(eventCred)
	for i := 0; i < len(eventSignatures); i++ {
		out = out + separator + base64.StdEncoding.EncodeToString(eventSignatures[i])
	}

	fmt.Fprint(ctx.Out, out)

	return nil
}

type verifyEventCredentialAction struct{}

func (a verifyEventCredentialAction) Execute(ctx node.Context) error {
	fmt.Println("*** action - Inside verifyEventCredential Action")
	var actor dkg.Actor

	err := ctx.Injector.Resolve(&actor)
	if err != nil {
		return xerrors.Errorf(resolveActorFailed, err)
	}

	idHash := ctx.Flags.String("idhash")
	eventName := ctx.Flags.String("eventName")
	eventCredentialString := ctx.Flags.String("eventCredential")
	eventCredential, err := base64.StdEncoding.DecodeString(eventCredentialString)
	if err != nil {
		return xerrors.Errorf("failed to decode event credential: %v", err)
	}
	eventSignaturesString := ctx.Flags.String("eventSignatures")
	eventSignaturesSplit := strings.Split(eventSignaturesString, ":")
	eventSignatures := [][]byte{}
	for i := 0; i < len(eventSignaturesSplit); i++ {
		thisEventSignature, err := base64.StdEncoding.DecodeString(eventSignaturesSplit[i])
		if err != nil {
			return xerrors.Errorf("failed to decode event signature: %v", err)
		}
		eventSignatures = append(eventSignatures, thisEventSignature)
	}

	verified, err := actor.VerifyEventCredential(idHash, eventName, eventCredential, eventSignatures)
	if err != nil {
		return xerrors.Errorf("failed to issue event credential: %v", err)
	}

	fmt.Fprint(ctx.Out, verified)

	return nil
}

type encryptAction struct{}

func (a encryptAction) Execute(ctx node.Context) error {
	var actor dkg.Actor

	err := ctx.Injector.Resolve(&actor)
	if err != nil {
		return xerrors.Errorf(resolveActorFailed, err)
	}

	message, err := hex.DecodeString(ctx.Flags.String("message"))
	if err != nil {
		errMessage := fmt.Sprintf("failed to decode message: %v", err)
		return xerrors.Errorf(errMessage)
	}

	k, c, remainder, err := actor.Encrypt(message)
	if err != nil {
		errMessage := fmt.Sprintf("failed to encrypt: %v", err)
		return xerrors.Errorf(errMessage)
	}

	outStr, err := encodeEncrypted(k, c, remainder)
	if err != nil {
		errMessage := fmt.Sprintf("failed to encode encrypted: %v", err)
		return xerrors.Errorf(errMessage)
	}

	fmt.Fprint(ctx.Out, outputEncryptSuccess(outStr))

	return nil
}

func outputEncryptSuccess(encrypted string) string {
	return fmt.Sprintf("ENCRYPT;success;%s", encrypted)
}

type decryptAction struct{}

func (a decryptAction) Execute(ctx node.Context) error {
	var actor dkg.Actor

	err := ctx.Injector.Resolve(&actor)
	if err != nil {
		errMessage := fmt.Sprintf("failed to resolve actor: %v", err)
		return xerrors.Errorf(errMessage)
	}

	encrypted := ctx.Flags.String("encrypted")

	k, c, err := decodeEncrypted(encrypted)
	if err != nil {
		errMessage := fmt.Sprintf("failed to decode encrypted str: %v", err)
		return xerrors.Errorf(errMessage)
	}

	decrypted, err := actor.Decrypt(k, c)
	if err != nil {
		errMessage := fmt.Sprintf("failed to decrypt: %v", err)
		return xerrors.Errorf(errMessage)
	}

	outputString := hex.EncodeToString(decrypted)
	fmt.Fprint(ctx.Out, outputDecryptSuccess(outputString))

	return nil
}

func outputDecryptSuccess(decrypted string) string {
	return fmt.Sprintf("DECRYPT;success;%s", decrypted)
}

func encodeEncrypted(k, c kyber.Point, remainder []byte) (string, error) {
	kbuff, err := k.MarshalBinary()
	if err != nil {
		return "", xerrors.Errorf("failed to marshal k: %v", err)
	}

	cbuff, err := c.MarshalBinary()
	if err != nil {
		return "", xerrors.Errorf("failed to marshal c: %v", err)
	}

	encoded := hex.EncodeToString(kbuff) + separator +
		hex.EncodeToString(cbuff) + separator +
		hex.EncodeToString(remainder)

	return encoded, nil
}

func decodeEncrypted(str string) (k kyber.Point, c kyber.Point, err error) {
	parts := strings.Split(str, separator)
	if len(parts) < 2 {
		return nil, nil, xerrors.Errorf("malformed encoded: %s", str)
	}

	// Decode K
	kbuff, err := hex.DecodeString(parts[0])
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to decode k point: %v", err)
	}

	k = suite.Point()

	err = k.UnmarshalBinary(kbuff)
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to unmarshal k point: %v", err)
	}

	// Decode C
	cbuff, err := hex.DecodeString(parts[1])
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to decode c point: %v", err)
	}

	c = suite.Point()

	err = c.UnmarshalBinary(cbuff)
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to unmarshal c point: %v", err)
	}

	return k, c, nil
}

// Verifiable encryption

type verifiableEncryptAction struct{}

func (a verifiableEncryptAction) Execute(ctx node.Context) error {
	var actor dkg.Actor

	err := ctx.Injector.Resolve(&actor)
	if err != nil {
		return xerrors.Errorf(resolveActorFailed, err)
	}

	// Decode GBar
	gBarbuff, err := hex.DecodeString(ctx.Flags.String("GBar"))
	if err != nil {
		return xerrors.Errorf("failed to decode GBar point: %v", err)
	}

	gBar := suite.Point()

	err = gBar.UnmarshalBinary(gBarbuff)
	if err != nil {
		return xerrors.Errorf("failed to unmarshal GBar point: %v", err)
	}

	// Decode the message
	message, err := hex.DecodeString(ctx.Flags.String("message"))
	if err != nil {
		return xerrors.Errorf("failed to decode message: %v", err)
	}

	ciphertext, remainder, err := actor.VerifiableEncrypt(message, gBar)
	if err != nil {
		return xerrors.Errorf("failed to encrypt: %v", err)
	}

	// Encoding the ciphertext
	// Encoding K
	kbuff, err := ciphertext.K.MarshalBinary()
	if err != nil {
		return xerrors.Errorf("failed to marshal k: %v", err)
	}

	// Encoding C
	cbuff, err := ciphertext.C.MarshalBinary()
	if err != nil {
		return xerrors.Errorf("failed to marshal c: %v", err)
	}

	// Encoding Ubar
	uBarbuff, err := ciphertext.UBar.MarshalBinary()
	if err != nil {
		return xerrors.Errorf("failed to marshal Ubar: %v", err)
	}

	// Encoding E
	ebuff, err := ciphertext.E.MarshalBinary()
	if err != nil {
		return xerrors.Errorf("failed to marshal E: %v", err)
	}

	// Encoding F
	fbuff, err := ciphertext.F.MarshalBinary()
	if err != nil {
		return xerrors.Errorf("failed to marshal F: %v", err)
	}

	outStr := hex.EncodeToString(kbuff) + separator +
		hex.EncodeToString(cbuff) + separator +
		hex.EncodeToString(uBarbuff) + separator +
		hex.EncodeToString(ebuff) + separator +
		hex.EncodeToString(fbuff) + separator +
		hex.EncodeToString(remainder)

	fmt.Fprint(ctx.Out, outStr)

	return nil
}

// Verifiable decrypt

type verifiableDecryptAction struct{}

func (a verifiableDecryptAction) Execute(ctx node.Context) error {
	var actor dkg.Actor

	err := ctx.Injector.Resolve(&actor)
	if err != nil {
		return xerrors.Errorf(resolveActorFailed, err)
	}

	// Decode GBar
	gBarbuff, err := hex.DecodeString(ctx.Flags.String("GBar"))
	if err != nil {
		return xerrors.Errorf("failed to decode GBar point: %v", err)
	}

	gBar := suite.Point()

	err = gBar.UnmarshalBinary(gBarbuff)
	if err != nil {
		return xerrors.Errorf("failed to unmarshal GBar point: %v", err)
	}

	// Decode the ciphertexts
	var ciphertextSlice []mTypes.Ciphertext

	ciphertextString := ctx.Flags.String("ciphertexts")

	parts := strings.Split(ciphertextString, separator)
	if len(parts)%5 != 0 {
		return xerrors.Errorf("malformed encoded: %s", ciphertextString)
	}

	batchSize := len(parts) / 5

	for i := 0; i < batchSize; i++ {

		// Decode K
		kbuff, err := hex.DecodeString(parts[i*5])
		if err != nil {
			return xerrors.Errorf("failed to decode k point: %v", err)
		}

		k := suite.Point()

		err = k.UnmarshalBinary(kbuff)
		if err != nil {
			return xerrors.Errorf("failed to unmarshal k point: %v", err)
		}

		// Decode C
		cbuff, err := hex.DecodeString(parts[i*5+1])
		if err != nil {
			return xerrors.Errorf("failed to decode c point: %v", err)
		}

		c := suite.Point()

		err = c.UnmarshalBinary(cbuff)
		if err != nil {
			return xerrors.Errorf("failed to unmarshal c point: %v", err)
		}

		// Decode UBar
		uBarbuff, err := hex.DecodeString(parts[i*5+2])
		if err != nil {
			return xerrors.Errorf("failed to decode UBar point: %v", err)
		}

		uBar := suite.Point()

		err = uBar.UnmarshalBinary(uBarbuff)
		if err != nil {
			return xerrors.Errorf("failed to unmarshal UBar point: %v", err)
		}

		// Decode E
		ebuff, err := hex.DecodeString(parts[i*5+3])
		if err != nil {
			return xerrors.Errorf("failed to decode E: %v", err)
		}

		e := suite.Scalar()

		err = e.UnmarshalBinary(ebuff)
		if err != nil {
			return xerrors.Errorf("failed to unmarshal E: %v", err)
		}

		// Decode F
		fbuff, err := hex.DecodeString(parts[i*5+4])
		if err != nil {
			return xerrors.Errorf("failed to decode F: %v", err)
		}

		f := suite.Scalar()

		err = f.UnmarshalBinary(fbuff)
		if err != nil {
			return xerrors.Errorf("failed to unmarshal F: %v", err)
		}

		ciphertextStruct := mTypes.Ciphertext{
			K:    k,
			C:    c,
			UBar: uBar,
			E:    e,
			F:    f,
			GBar: gBar,
		}

		ciphertextSlice = append(ciphertextSlice, ciphertextStruct)

	}

	decrypted, err := actor.VerifiableDecrypt(ciphertextSlice)
	if err != nil {
		return xerrors.Errorf("failed to decrypt: %v", err)
	}

	var decryptString []string

	for i := 0; i < batchSize; i++ {
		decryptString = append(decryptString, hex.EncodeToString(decrypted[i]))
	}

	fmt.Fprint(ctx.Out, decryptString)
	return nil
}

// reshare

type reshareAction struct{}

func (a reshareAction) Execute(ctx node.Context) error {
	var actor dkg.Actor

	err := ctx.Injector.Resolve(&actor)
	if err != nil {
		return xerrors.Errorf(resolveActorFailed, err)
	}

	co, _, err := getCollectiveAuth(ctx)
	if err != nil {
		return xerrors.Errorf("failed to get collective authority: %v", err)
	}

	t := ctx.Flags.Int("thresholdNew")

	err = actor.Reshare(co, t)
	if err != nil {
		return xerrors.Errorf("failed to reshare: %v", err)
	}

	fmt.Fprintf(ctx.Out, "✅ Reshare done.\n")

	return nil
}
