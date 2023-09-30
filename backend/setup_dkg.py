import os
import subprocess
import shlex
import time
from subprocess import call
import threading
import sys
import json

def build_cli_dkg():
    '''
    Builds dkg cli
    '''
    command = f'go install .'
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')
    time.sleep(2)

def setup_dkg_node(port, node):
    '''
    Sets up DKG node
    '''
    proc = subprocess.Popen(f"LLVL=info dkgcli --config /tmp/node{node} start --routing tree --listen tcp://127.0.0.1:{port}", shell=True, stdout=subprocess.PIPE)
    while True:
        data = proc.stdout.readline()   # Alternatively proc.stdout.read(1024)
        if len(data) != 0:
            # break
            data_decode = data.decode('utf-8')
            sys.stdout.write(data_decode)   # sys.stdout.buffer.write(data) on Python 3.x
            f = open("outputs/dkg_outputs.txt", "a")
            f.write(data_decode)
            sys.stdout.flush()

def setup_dkg_nodes(num_nodes, start_port, start_node):
    open('outputs/dkg_outputs.txt', 'w').close()
    threads = []
    for i in range(0, num_nodes):
        threads.append(threading.Thread(target=setup_dkg_node, args=(start_port + i, start_node + i)))

    for i in range(0, num_nodes):
        threads[i].start()
    
    time.sleep(1)

def setup_dkg(start_node):
    command = f'''\
        dkgcli --config /tmp/node{start_node} dkg setup \
        --authority $(cat /tmp/node{start_node}/dkgauthority) --authority $(cat /tmp/node{start_node + 1}/dkgauthority) --authority $(cat /tmp/node{start_node + 2}/dkgauthority)\
        -- numKeys 2 --numUsers 5
        '''
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')

def share_certificates(start_node, start_port):
    print("*** Sharing certificates")
    # Share Certificates
    # Share between nodes 1 and 2
    print("Sharing certificates between nodes 1 and 2")
    command = f'dkgcli --config /tmp/node{start_node + 1} minogrpc join --address //127.0.0.1:{start_port} $(dkgcli --config /tmp/node{start_node} minogrpc token)'
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')
    time.sleep(1)

    # Share between nodes 1 and 3
    print("Sharing certificates between nodes 1 and 3")
    command = f'dkgcli --config /tmp/node{start_node + 2} minogrpc join --address //127.0.0.1:{start_port} $(dkgcli --config /tmp/node{start_node} minogrpc token)'
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')
    time.sleep(1)

def init_dkg(start_node):
    print("Initialize dkg on node 1")
    command = f'dkgcli --config /tmp/node{start_node} dkg listen'
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')
    time.sleep(1)

    print("Initialize dkg on node 2")
    command = f'dkgcli --config /tmp/node{start_node+1} dkg listen'
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')
    time.sleep(1)

    print("Initialize dkg on node 3")
    command = f'dkgcli --config /tmp/node{start_node+2} dkg listen'
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')
    time.sleep(1)

def start_dkg(start_node, start_port):
    setup_dkg(start_node)
    time.sleep(1)
    share_certificates(start_node, start_port)
    time.sleep(1)
    init_dkg(start_node)
    time.sleep(1)
    setup_dkg(start_node)
    time.sleep(1)

if __name__ == "__main__":
    try:
        print(os.getcwd())
        print(os.path.isfile('setup_info.json'))
        with open('setup_info.json', 'r') as f:
            start_info = json.load(f)
            print("Start info: ", start_info)
            if "num_nodes_dkg" in start_info and "start_node_dkg" in start_info and "start_port_dkg" in start_info:
                num_nodes_dkg = start_info['num_nodes_dkg']
                start_port_dkg = start_info['start_port_dkg']
                start_node_dkg = start_info['start_node_dkg']

                os.chdir("dela/dkg/pedersen/dkgcli")
                build_cli_dkg()
                setup_dkg_nodes(num_nodes_dkg, start_port_dkg, start_node_dkg)
                start_dkg(start_node_dkg, start_port_dkg)
                
    except Exception as e:
        print("setup_info.json must contain num_nodes_dkg, start_node_dkg, and start_port_dkg")