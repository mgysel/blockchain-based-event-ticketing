import os
import subprocess
import shlex
import time
import appscript
from subprocess import call

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
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    masterCredential = process.stdout.read().decode('utf-8')
    print(f'Process output: {masterCredential}')

    return masterCredential

def issue_event_credential(start_node, master_credential):
    print(f'Issuing event credential for {master_credential}')
    command = f'dkgcli --config /tmp/node{start_node} dkg issueEventCredential --masterCredential {master_credential}'
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    eventCredential = process.stdout.read().decode('utf-8')
    print(f'Process output: {eventCredential}')

    return eventCredential