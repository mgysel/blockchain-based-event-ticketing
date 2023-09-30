import sys
import os

from flask import Flask, request, redirect, url_for, make_response, jsonify
from json import dumps
from werkzeug.security import generate_password_hash, check_password_hash
from bson.objectid import ObjectId
import subprocess
import time

######################################################
##################### WRITE ##########################
######################################################

def value_write_server(data, start_node_dela):
    '''
    Writes key-value pair to dela value smart contract
    '''
    print("*** Inside value_write_server")
    print("Data: ", data)
    fields = ['key','value']
    for field in fields:
        if not field in data:
            return make_response(
                dumps(
                    {
                        "message": "Write requires key and value fields.",
                        "data": {}
                    }
                ),
                400
            ) 
    
    key = data['key']
    value = data['value']

    # Submit write tx to blockchain
    response = value_write(start_node_dela, key, value)
    
    if response['message'] == "Error.":
        return make_response(dumps(response), 400)
    
    return make_response(dumps(response), 201)

def value_write(start_node_dela, key, value):
    '''
    Writes value to dela value smart contract
    '''
    value_contract_write(start_node_dela, key, value)
    response = value_contract_write_response(key, value)
    print("Write Response: ", response)

    return response

def value_contract_write(start_node, key, value):
    print("*** Store a value on the value contract.")
    pk_path = os.getcwd() + '/dela/private.key'
    command = f'''
        memcoin \
            --config /tmp/node{start_node} pool add \
            --key {pk_path} \
            --args go.dedis.ch/dela.ContractArg --args go.dedis.ch/dela.Value \
            --args value:command --args WRITE \
            --args value:key --args {key} \
            --args value:value --args {value} \
        '''
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    time.sleep(1)
    return process_output

def value_contract_write_response(key, value):
    '''
    Returns response from blockchain of a value contract write command 
    '''
    print("*** Inside value_contract_write_response")
    readlines = []
    fn = f'{os.getcwd()}/dela/outputs/dela_outputs.txt'
    with open(fn, 'r') as f:
        for line in f:
            if "//VALUECONTRACT_WRITEOUTPUT" in line and str(key) in line and str(value) in line:
                readlines.append(line)
                print("Have line with key/value: ", line)

    readlines.reverse()
    print("Readlines: ", readlines)
    if (len(readlines) > 0):
        line = readlines[0].split("//")[1]
        split_line = line.split(";")
        print("Write response Split line: ", split_line)
        print("Length of Split line: ", len(split_line))

        # Success
        if split_line[1] == "success" and len(split_line) == 4:
            read_key = split_line[2]
            read_val = split_line[3]
            if read_key == key and read_val == value:
                print("key-val match")
                return {
                    "message": "Success.",
                    "data": {
                        "key": read_key,
                        "value": read_val,
                    }
                }
            # Error
            elif split_line[1] == "error" and len(split_line) == 4:
                message = split_line[2]
                return {
                    "message": "Error.",
                    "data": message,
                }

    return {
        "message": "Error.",
        "data": "Write tx failed.",
    }

######################################################
###################### READ ##########################
######################################################

def value_read_server(args, start_node_dela):
    '''
    Reads a value from dela blockchain
    given key
    '''
    print("*** Inside value_read")
    key = args.get('key', None)
    response = value_read(key, start_node_dela)
    print("Read Response: ", response)

    return make_response(dumps(response), 201)

def value_read(key, start_node_dela):
    '''
    Reads a value from dela blockchain
    given key
    '''
    print("*** Inside value_read")
    # Submit read tx to blockchain, read response
    value_contract_read(start_node_dela, key)
    response = value_contract_read_response(key)
    print("Read Response: ", response)

    return response

def value_contract_read(start_node, key):
    '''
    Submits a read command to dela value smart contract
    '''
    print("*** Read a value on the value contract.")
    pk_path = os.getcwd() + '/dela/private.key'
    command = f'''
        memcoin \
        --config /tmp/node{start_node} pool add \
        --key {pk_path} \
        --args go.dedis.ch/dela.ContractArg --args go.dedis.ch/dela.Value \
        --args value:key --args {key} \
        --args value:command --args READ 
    '''
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')
    time.sleep(1)
    return process_output

def value_contract_read_response(key):
    '''
    Returns read response from dela value blockchain
    '''
    print("*** Inside value_read_text")
    readlines = []
    fn = os.getcwd() + '/dela/outputs/dela_outputs.txt'
    with open(fn, 'r') as f:
        for line in f:
            if "//VALUECONTRACT_READOUTPUT" in line and key in line:
                readlines.append(line)

    readlines.reverse()
    if (len(readlines) > 0):
        line = readlines[0].split("//")[1]
        split_line = line.split(";")

        # Success
        if split_line[1] == "success" and len(split_line) == 4:
            read_key = split_line[2]
            read_val = split_line[3]
            if read_key == key:
                return {
                    "message": "Success.",
                    "data": {
                        "key": read_key,
                        "value": read_val,
                    }
                }
            # Error
            elif split_line[1] == "error" and len(split_line) == 4:
                message = split_line[2]
                read_key = split_line[3]
                if read_key == key:
                    return {
                        "message": "Error.",
                        "data": message,
                    }

    return {
        "message": "Error.",
        "data": "Read tx failed.",
    }

######################################################
###################### LIST ##########################
######################################################

def value_list_server(start_node_dela):
    '''
    Reads all key-value pairs from dela blockchain value contract
    returns server response
    '''
    print("*** Inside value_list_server")

    response = value_list(start_node_dela)
    if response['message'] == "Error.":
        return make_response(dumps(response), 400)
    return make_response(dumps(response), 201)

def value_list(start_node_dela):
    '''
    Lists all values from dela blockchain value smart contract
    '''
    print("*** Inside value_list")
    value_contract_list_command(start_node_dela)
    response = value_contract_list_response()

    return response

def value_contract_list_command(start_node):
    '''
    Submits a list command to dela value smart contract
    '''
    print("*** List values on the value contract.")
    pk_path = os.getcwd() + '/dela/private.key'
    command = f'''
        memcoin \
        --config /tmp/node{start_node} pool add\
        --key {pk_path}\
        --args go.dedis.ch/dela.ContractArg --args go.dedis.ch/dela.Value\
        --args value:command --args LIST
    '''
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    time.sleep(1)
    print(f'Process output: {process_output}')

def value_contract_list_response():
    '''
    Returns list response from dela value blockchain
    '''
    print("*** Inside value_contract_list_response")
    readlines = []
    fn = os.getcwd() + '/dela/outputs/dela_outputs.txt'
    with open(fn, 'r') as f:
        for line in f:
            if "//VALUECONTRACT_LISTOUTPUT" in line:
                readlines.append(line)

    readlines.reverse()
    if (len(readlines) > 0):
        line = readlines[0].split("//")[1]
        split_line = line.split(";")
        print("Split line: ", split_line)

        # Success
        if split_line[1] == "success" and len(split_line) >= 3:
            pairs_split = split_line[2:]
            pairs_output = []
            for pair in pairs_split:
                pair_split = pair.split("=")
                pairs_output.append({
                    "key": pair_split[0],
                    "value": pair_split[1],
                })
            return {
                "message": "Success.",
                "data": {
                    "pairs": pairs_output,
                }
            }
        # Error
        elif split_line[1] == "error" and len(split_line) == 3:
            message = split_line[2]
            return {
                "message": "Error.",
                "data": message,
            }

    return {
        "message": "Error.",
        "data": "List tx failed.",
    }

######################################################
##################### DELETE #########################
######################################################

def value_delete_server(data, start_node_dela):
    '''
    Deletes a value from value contract
    given key
    '''
    print("*** Inside value_delete_server")
    print("Data: ", data)
    fields = ['key']
    for field in fields:
        if not field in data:
            return make_response(
                dumps(
                    {
                        "message": "Write requires key field.",
                        "data": {}
                    }
                ),
                400
            ) 
    
    key = data['key']

    # Submit write tx to blockchain
    response = value_delete(key, start_node_dela)
    print("Delete Response: ", response)
    
    if response['message'] == "Error.":
        return make_response(dumps(response), 400)
    
    return make_response(dumps(response), 201)

def value_delete(key, start_node_dela):
    '''
    Deletes a value from dela blockchain
    given key
    '''
    print("*** Inside value_delete")
    # Submit read tx to blockchain, read response
    value_contract_delete(start_node_dela, key)
    response = value_contract_delete_response(key)

    return response

def value_contract_delete(start_node, key):
    '''
    Submits a delete command to dela value smart contract
    '''
    print("*** Delete a value on the value contract.")
    print("Key: ", key)
    pk_path = os.getcwd() + '/dela/private.key'
    command = f'''
        memcoin \
        --config /tmp/node{start_node} pool add \
        --key {pk_path} \
        --args go.dedis.ch/dela.ContractArg --args go.dedis.ch/dela.Value \
        --args value:key --args {key} \
        --args value:command --args DELETE 
    '''
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    time.sleep(1)
    return process_output

def value_contract_delete_response(key):
    '''
    Returns delete response from dela value blockchain
    '''
    print("*** Inside value_delete_response")
    readlines = []
    fn = os.getcwd() + '/dela/outputs/dela_outputs.txt'
    with open(fn, 'r') as f:
        for line in f:
            if "//VALUECONTRACT_DELETEOUTPUT" in line and key in line:
                readlines.append(line)

    readlines.reverse()
    if (len(readlines) > 0):
        line = readlines[0].split("//")[1]
        split_line = line.split(";")
        print("Split line: ", split_line)

        # Success
        if split_line[1] == "success" and len(split_line) == 3:
            delete_key = split_line[2]
            if delete_key == key:
                return {
                    "message": "Success.",
                    "data": {
                        "key": delete_key,
                    }
                }
            # Error
            elif split_line[1] == "error" and len(split_line) == 4:
                message = split_line[2]
                delete_key = split_line[3]
                if delete_key == key:
                    return {
                        "message": "Error.",
                        "data": message,
                    }

    return {
        "message": "Error.",
        "data": "Delete tx failed.",
    }