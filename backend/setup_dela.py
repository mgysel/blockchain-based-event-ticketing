import sys
import os
import subprocess
import shlex
import time
from subprocess import call
import threading
import json
from routes.objects.MongoWrapper import MongoWrapper
from routes.routes.value import value_write

print("Setup_dela.py path: ", os.getcwd())

def build_cli_memcoin():
    '''
    Builds memcoin cli
    '''
    print("CWD 1: ", os.getcwd())
    os.chdir("cli/node/memcoin")
    print("CWD 2: ", os.getcwd())
    print("Sharing certificates between nodes 1 and 2")
    command = f'go install'
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')
    time.sleep(2)
    os.chdir("../../..")
    print("CWD 3: ", os.getcwd())

# def build_cli_dkg():

def setup_dela_node(port, node):
    '''
    Sets up DELA blockchain nodes
    '''
    proc = subprocess.Popen(f"LLVL=info memcoin --config /tmp/node{node} start --listen tcp://127.0.0.1:{port}", shell=True, stdout=subprocess.PIPE)
    while True:
        data = proc.stdout.readline()   # Alternatively proc.stdout.read(1024)
        if len(data) != 0:
            # break
            data_decode = data.decode('utf-8')
            sys.stdout.write(data_decode)   # sys.stdout.buffer.write(data) on Python 3.x
            f = open("outputs/dela_outputs.txt", "a")
            f.write(data_decode)
            sys.stdout.flush()

def setup_dela_nodes(num_nodes, start_port, start_node):
    open('outputs/dela_outputs.txt', 'w').close()
    threads = []
    for i in range(0, num_nodes):
        threads.append(threading.Thread(target=setup_dela_node, args=(start_port + i, start_node + i)))

    for i in range(0, num_nodes):
        threads[i].start()
    
    time.sleep(1)

def share_certificates(start_node, start_port):
    print("*** Sharing certificates")
    # Share Certificates
    # Share between nodes 1 and 2
    print("Sharing certificates between nodes 1 and 2")
    command = f'memcoin --config /tmp/node{start_node+1} minogrpc join --address //127.0.0.1:{start_port} $(memcoin --config /tmp/node{start_node} minogrpc token)'
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')
    time.sleep(1)

    # Share between nodes 1 and 3
    print("Sharing certificates between nodes 1 and 3")
    command = f'memcoin --config /tmp/node{start_node+2} minogrpc join --address //127.0.0.1:{start_port} $(memcoin --config /tmp/node{start_node} minogrpc token)'
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')
    time.sleep(1)

def create_chain(start_node):
    print("*** Creating a chain with nodes")
    # Create a new chain with the three nodes
    command = f'memcoin --config /tmp/node{start_node} ordering setup --member $(memcoin --config /tmp/node{start_node} ordering export) --member $(memcoin --config /tmp/node{start_node+1} ordering export) --member $(memcoin --config /tmp/node{start_node+2} ordering export)'
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')

def create_bls_signer():
    print("*** Create a bls signer to sign transactions.")
    command = 'crypto bls signer new --save private.key --force'
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')

    command = 'crypto bls signer read --path private.key --format BASE64'
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')

def authorize_signer(start_node):
    print("*** Authorize the signer to handle the access contract on each node")
    command = f'memcoin --config /tmp/node{start_node} access add --identity $(crypto bls signer read --path private.key --format BASE64_PUBKEY)'
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')
    time.sleep(1)

    command = f'memcoin --config /tmp/node{start_node+1} access add --identity $(crypto bls signer read --path private.key --format BASE64_PUBKEY)'
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')
    time.sleep(1)

    command = f'memcoin --config /tmp/node{start_node+2} access add --identity $(crypto bls signer read --path private.key --format BASE64_PUBKEY)'
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')
    time.sleep(1)

def setup_blockchain(start_node, start_port):
    share_certificates(start_node, start_port)
    time.sleep(1)
    create_chain(start_node)
    time.sleep(1)
    create_bls_signer()
    time.sleep(1)
    authorize_signer(start_node)
    time.sleep(1)

# ALlow contracts
def allow_value_contract(start_node):
    print("*** Update the access contract to allow us to use the value contract.")
    pk_path = os.getcwd() + '/private.key'
    command = f'''
        memcoin \
            --config /tmp/node{start_node} pool add \
            --key {pk_path} \
            --args go.dedis.ch/dela.ContractArg --args go.dedis.ch/dela.Access \
            --args access:grant_id --args 0200000000000000000000000000000000000000000000000000000000000000 \
            --args access:grant_contract --args go.dedis.ch/dela.Value \
            --args access:grant_command --args all \
            --args access:identity --args $(crypto bls signer read --path {pk_path} --format BASE64_PUBKEY) \
            --args access:command --args GRANT
    '''
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')
    time.sleep(1)
    return process_output

def allow_event_contract(start_node):
    '''
    Sends tx to blockchain to allow event smart contract
    '''
    print("*** Inside event_contract_allow_command.")
    print("CWD: ", os.getcwd())
    print("Dir exists? ", os.path.isdir(os.getcwd()))
    print("Private.key exists? ", os.path.isfile(os.getcwd() + '/dela/private.key'))
    pk_path = os.getcwd() + '/private.key'
    command = f'''
        memcoin \
            --config /tmp/node{start_node} pool add \
            --key {pk_path} \
            --args go.dedis.ch/dela.ContractArg --args go.dedis.ch/dela.Access \
            --args access:grant_id --args 0300000000000000000000000000000000000000000000000000000000000000 \
            --args access:grant_contract --args go.dedis.ch/dela.Event \
            --args access:grant_command --args all \
            --args access:identity --args $(crypto bls signer read --path {pk_path} --format BASE64_PUBKEY) \
            --args access:command --args GRANT
    '''
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Event contract allow Process output: {process_output}')
    time.sleep(1)
    return process_output

def value_contract_write(start_node, key, value):
    '''
    Stores a value on the value contract
    '''
    print("*** Store a value on the value contract.")
    pk_path = os.getcwd() + '/private.key'
    command = f'''
        memcoin \
            --config /tmp/node{start_node} pool add \
            --key {pk_path} \
            --args go.dedis.ch/dela.ContractArg --args go.dedis.ch/dela.Value \
            --args value:key --args {key} \
            --args value:value --args {value} \
            --args value:command --args WRITE
        '''
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')
    time.sleep(1)
    return process_output

########################################################################################################################
#################################################### TESTING ###########################################################
########################################################################################################################

if __name__ == "__main__":
    try:
        # Reset database tables
        db = MongoWrapper().client['ticketing_store']
        db.drop_collection('events')
        db.drop_collection('users')
        db.create_collection('events')
        db.create_collection('users')

        # Startup dela
        print(os.getcwd())
        print(os.path.isfile('setup_info.json'))
        with open('setup_info.json', 'r') as f:
            start_info = json.load(f)
            print("Start info: ", start_info)
            if "num_nodes_dela" in start_info and "start_node_dela" in start_info and "start_port_dela" in start_info:
                num_nodes_dela = start_info['num_nodes_dela']
                start_port_dela = start_info['start_port_dela']
                start_node_dela = start_info['start_node_dela']

                print("CWD BEFORE 1: ", os.getcwd())
                os.chdir("dela")
                print("CWD AFTER 1: ", os.getcwd())
                
                # Setup dela
                build_cli_memcoin()
                setup_dela_nodes(num_nodes_dela, start_port_dela, start_node_dela)
                setup_blockchain(start_node_dela, start_port_dela)
                allow_value_contract(start_node_dela)
                allow_event_contract(start_node_dela)

                # Initialize 5 accounts with $10,000
                for i in range(0, 5):
                    value_contract_write(start_node_dela, f"bank:account:{i}", '10000')
                
    except Exception as e:
        print("setup_info.json must contain num_nodes_dela, start_node_dela, and start_port_dela")
