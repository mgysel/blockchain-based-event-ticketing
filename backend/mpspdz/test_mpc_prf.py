import os
import subprocess
import random
from pyseltongue import SecretSharer, PlaintextToHexSecretSharer
import hashlib
import string
import timeit

# Create shares from name
def create_name_shares(name, num_players):
    '''
    Creates secret shares for a given name
    Outputs the shares in files Player-Data/Input-P<i>-0
    '''
    print("*** Inside create_name_shares")
    # Determine sha256 hash of name
    m = hashlib.sha256()
    m.update(name.encode('utf-8'))
    hash_name = m.hexdigest()

    # Determine secret shares of name
    shares = SecretSharer.split_secret(hash_name, num_players, num_players)
    print(f'Shares: {shares}')

    # Format secret shares of name to integers
    share_ints = []
    for share in shares:
        share_hex = share.split('-')[1]
        share_int = int(share_hex, 16)
        print(f'This Share: {share_int}')
        share_ints.append(share_int)

    # Store secret shares in files Player-Data/Input-P<i>-0
    for i in range(len(share_ints)):
        path = os.path.join(os.path.dirname(__file__), f'Player-Data/Input-P{i}-0')
        print("I: ", i)
        print("PATH: ", path)
        
        with open(path, 'w') as f:
            f.write(str(share_ints[i]))
        
        f.close()

    return shares

def run_mpc_hash_time(num_players):
    '''
    Uses MP-SPDZ mpc to determine the mimc prf of secret shared name
    Secret shared name stored in files Player-Data/Input-P<i>-0
    Key and num_rounds must be specified for the mimc prf function
    Returns the hash of the name
    '''
    k = random.randint(0, 2**128)
    num_rounds = 130

    # Compile and run prf_mimc_mine.mpc with num_players
    print("This location: ", os.path.dirname(__file__))
    path = os.path.join(os.path.dirname(__file__), 'Scripts/compile-run.py')
    print("MPC HASH PATH: ", path)
    command = f'{path} -E mascot prf_mimc_mine {num_players} {num_rounds} {k}'
    print("Command: ", command)
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    
    # Wait to finish and receive output
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')
    
    # Obtain hash from output
    process_output_array = process_output.split('\n')
    a = [el for el in process_output_array if 'Time = ' in el]
    if len(a) > 0:
        time_taken = float(a[0].split('Time = ')[1].split(' ')[0].strip())
        print("Time Taken: ", time_taken)
        return time_taken

    return None

def test_latency_mimc(num_parties, num_rounds):
    '''
    Tests the latency of the mimc prf function
    num_parties also has to be updated in mpspdz/Scripts/run-common.sh on line 82
        players=${PLAYERS:-num_parties}
    '''
    time_taken = []
    for i in range(0, num_rounds):
        name = ''.join(random.choice(string.ascii_letters) for i in range(50))
        create_name_shares(name, num_parties)
        this_time_taken = run_mpc_hash_time(num_parties)
        time_taken.append(this_time_taken)
    
    print("Average time taken: ", sum(time_taken)/len(time_taken))

test_latency_mimc(2, 15)