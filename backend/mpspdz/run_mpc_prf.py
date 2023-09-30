import os
import subprocess
import random
from pyseltongue import SecretSharer, PlaintextToHexSecretSharer
import hashlib

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

def run_mpc_hash(num_players, key, num_rounds):
    '''
    Uses MP-SPDZ mpc to determine the mimc prf of secret shared name
    Secret shared name stored in files Player-Data/Input-P<i>-0
    Key and num_rounds must be specified for the mimc prf function
    Returns the hash of the name
    '''
    # Compile and run prf_mimc_mine.mpc with num_players
    print("This location: ", os.path.dirname(__file__))
    path = os.path.join(os.path.dirname(__file__), 'Scripts/compile-run.py')
    print("MPC HASH PATH: ", path)
    command = f'{path} -E mascot prf_mimc_mine {num_players} {num_rounds} {key}'
    print("Command: ", command)
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    
    # Wait to finish and receive output
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    
    # Obtain hash from output
    process_output_array = process_output.split('\n')
    a = [el for el in process_output_array if 'Encrypted Message: ' in el]
    if len(a) > 0:
        h = a[0].split('Encrypted Message: ')[1]
        return h

    return None

def determine_hash():
    '''
    Determines the hash of a given name
    '''
    num_players = 2
    # key = random.getrandbits(128)
    key = 123456789
    num_rounds = 5

    h = run_mpc_hash(num_players, key, num_rounds)
    return h

# Create name shares
name = 'MikeGysel'
num_players = 2
create_name_shares(name, num_players)
h1 = determine_hash()
print(f'Hash 1: {h1}')

name = 'MikeGysel'
num_players = 2
create_name_shares(name, num_players)
h2 = determine_hash()
print(f'Hash 2: {h2}')
