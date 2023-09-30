import os
import subprocess
import random
from pyseltongue import SecretSharer, PlaintextToHexSecretSharer
import hashlib

###############################################################
######################### MPC HASH ############################
###############################################################

def determine_mpc_hash(name, num_players, key, num_rounds):
    '''
    Determines the hash of a given name
    '''
    # Create secret shares of name 
    create_name_shares(name, num_players)

    # Run mpc hash
    h = run_mpc_hash(num_players, key, num_rounds)

    return h

# Create shares from name
def create_name_shares(name, num_players):
    '''
    Creates secret shares for a given name
    Outputs the shares in files Player-Data/Input-P<i>-0
    '''
    print("Inside create_name_shares: ")
    print("Name: ", name)

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
        share_ints.append(share_int)

    # Store secret shares in files Player-Data/Input-P<i>-0
    for i in range(len(share_ints)):
        path = os.path.join(os.getcwd(), f'mpspdz/Player-Data/Input-P{i}-0')
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
    print("Current directory: ", os.getcwd())
    os.chdir('mpspdz')
    print("Current directory after change: ", os.getcwd())

    # Compile and run prf_mimc_mine.mpc with num_players
    path = os.path.join(os.getcwd(), 'Scripts/compile-run.py')
    command = f'{path} -E mascot prf_mimc_mine {num_players} {num_rounds} {key}'
    print("Command: ", command)
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    
    # Wait to finish and receive output
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')

    os.chdir('../')
    
    # Obtain hash from output
    process_output_array = process_output.split('\n')
    encrypted_message = None
    a = [el for el in process_output_array if 'Encrypted Message: ' in el]
    if len(a) > 0:
        encrypted_message = a[0].split('Encrypted Message: ')[1].replace(' ', '').replace('-', '').replace('[', '').replace(']', '').replace(',', '')
        return encrypted_message

    return encrypted_message