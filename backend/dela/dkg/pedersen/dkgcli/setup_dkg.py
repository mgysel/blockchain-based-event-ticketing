import os
import subprocess
import shlex
import time
import appscript
from subprocess import call

def setup_dkg_nodes(num_nodes, start_port, start_node):
    '''
    Sets up DKG nodes
    '''
    print("*** Setting up DELA Nodes")
    for i in range(0, num_nodes):

        # New terminals
        t = appscript.app('Terminal')

        # Setup 3 nodes in 3 terminals
        # Node 1
        print(f"Starting node {i+1}")
        cwd = os.getcwd()
        print("cwd: ", cwd)
        command1 = f'cd {cwd}'
        command2 = f'LLVL=info dkgcli --config /tmp/node{start_node + i} start --routing tree --listen tcp://127.0.0.1:{start_port + i}'
        # subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
        t.do_script(f'{command1}; {command2}')
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

def encrypt(start_node, pt):
    ptHex = pt.encode('utf-8').hex()
    command = f'''
        dkgcli --config /tmp/node{start_node} dkg encrypt --message {ptHex}
    '''
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    ct = process.stdout.read().decode('utf-8')
    print(f'Process output: {ct}')

    return ct

def decrypt(start_node, ct):
    command = f'''
        dkgcli --config /tmp/node{start_node} dkg decrypt --encrypted {ct}
    '''
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    ptHex = process.stdout.read().decode('utf-8')
    pt = bytes.fromhex(ptHex).decode('utf-8')
    print(f'Process output: {pt}')

    return pt

def issue_master_credential(start_node, id_hash):
    print(f'Issuing master credential for {id_hash}')
    command = f'dkgcli --config /tmp/node{start_node} dkg issueMasterCredential --idhash {id_hash}'
    print("Command: ", command)
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    masterCredential = process.stdout.read().decode('utf-8')
    print(f'Process output: {masterCredential}')
    time.sleep(1)

    return masterCredential

def issue_event_credential(start_node, id_hash, event_name, master_credential, master_signatures):
    print(f'Issuing event credential for {master_credential}')
    command = f'dkgcli --config /tmp/node{start_node} dkg issueEventCredential --idhash {id_hash} --eventName {event_name} --masterCredential {master_credential} --masterSignatures {master_signatures}'
    print("Command: ", command)
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    eventCredential = process.stdout.read().decode('utf-8')
    print(f'Process output: {eventCredential}')
    time.sleep(1)

    return eventCredential

def verify_event_credential(start_node, id_hash, event_name, event_credential, event_signatures):
    print(f'Verifying event credential for {event_credential}')
    command = f'dkgcli --config /tmp/node{start_node} dkg verifyEventCredential --idhash {id_hash} --eventName {event_name} --eventCredential {event_credential} --eventSignatures {event_signatures}'
    print("Command: ", command)
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    eventCredential = process.stdout.read().decode('utf-8')
    print(f'Process output: {eventCredential}')
    time.sleep(1)

    return eventCredential

start_node = 36
start_port = 2036
setup_dkg_nodes(3, start_port, start_node)
start_dkg(start_node, start_port)

# message = "hello"
# print("Message: ", message)
# e = encrypt(start_node, message)
# print("Encrypted: ", e)
# d = decrypt(start_node, e)
# print("Decrypted: ", d)

# id_hash = "MichaelGyselHash6"
# mc = issue_master_credential(start_node, id_hash)
# print("Master credential: ", mc)

# event_name = "EventName"
# mc_string_components = mc.split(":")
# master_credential = mc_string_components[0]
# master_signatures = mc_string_components[1:]

# print("master_credential: ", master_credential)
# print("master_signatures: ", master_signatures)
# print("master_signatures join: ", ":".join(master_signatures))

# ec = issue_event_credential(start_node, id_hash, event_name, master_credential, ":".join(master_signatures))
# print("Event credential: ", ec)

# ec_string_components = ec.split(":")
# event_credential = ec_string_components[0]
# event_signatures = ec_string_components[1:]

# print("event_credential: ", event_credential)
# print("event_signatures: ", event_signatures)
# print("event_signatures join: ", ":".join(event_signatures))
# verified = verify_event_credential(start_node, id_hash, event_name, event_credential, ":".join(event_signatures))
# print("Verified? ", verified)