from flask import make_response
from json import dumps
import subprocess
import time 
import random

from routes.objects.userObject import User
from routes.routes.mpc import determine_mpc_hash

###############################################################
########################## ENCRYPT ############################
###############################################################

def dkg_encrypt_message_server(user_id, start_node, data):
    '''
    Encrypts message using DKG
    Returns HTTP response for server
    '''
    fields = ['message']
    for field in fields:
        if not field in data:
            return make_response(
                dumps(
                    {
                        "message": "Encrypt requires message.",
                        "data": {}
                    }
                ),
                400
            )

    # Submit encrypt message to dkg
    message = data['message']
    response = dkg_encrypt_message(start_node, message)

    # If error, return response
    if response['message'] == "Error.":
        return make_response(dumps(response), 400)
    return make_response(dumps(response), 201)

def dkg_encrypt_message(start_node, message):
    '''
    Encrypts message using DKG
    '''

    # Split message into 24 characters 
    # 24 characters is the max length of a message
    split_message = [message[i:i+24] for i in range(0, len(message), 24)]
    encrypted_split_message = []
    for sm in split_message:
        # Submit encrypt message to dkg
        response = dkg_encrypt_command(start_node, sm)
        parsed_response = dkg_encrypt_response(response)

        # Get parsed response, add encrypted message to encrypted_split_message
        if parsed_response['message'] == "Error.":
            return parsed_response
        encrypted_split_message.append(parsed_response['data']['encrypted_message'].strip())

    # Combine encrypted_split_message into one encrypted message
    encrypted_message = "/".join(encrypted_split_message)
    result = {
        "message": "Success.",
        "data": {
            "encrypted_message": encrypted_message,
        }
    }
    return result

def dkg_encrypt_command(start_node, pt):
    '''
    Submits dkg encrypt to cli, returns response
    '''
    ptHex = pt.encode('utf-8').hex()
    command = f'''
        dkgcli --config /tmp/node{start_node} dkg encrypt --message {ptHex}
    '''
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    ct = process.stdout.read().decode('utf-8')

    return ct

def dkg_encrypt_response(response):
    '''
    Reads response from dkg encrypt command
    Parses response
    '''
    response_split = response.split(";")
    if len(response_split) != 3:
        return {
            "message": "Error.",
            "data": "DKG failed to encrypt."
        }
    
    if response_split[0] != 'ENCRYPT':
        return {
            "message": "Error.",
            "data": "DKG failed to encrypt."
        }
    
    if response_split[0] == 'ENCRYPT' and response_split[1] == "error":
        return {
            "message": "Error.",
            "data": response_split[2],
        }
    
    if response_split[0] == 'ENCRYPT' and response_split[1] == "success":
        ct = response_split[2]
        return {
            "message": "Success.",
            "data": {
                "encrypted_message": ct,
            }
        }

###############################################################
########################## DECRYPT ############################
###############################################################

def dkg_decrypt_message_server(user_id, start_node, data):
    '''
    Decrypts encrypted message using DKG
    Gets response for server
    '''
    fields = ['encrypted_message']
    for field in fields:
        if not field in data:
            return make_response(
                dumps(
                    {
                        "message": "Decrypt requires encrypted message.",
                        "data": {}
                    }
                ),
                400
            )

    # Submit buy ticket to blockchain
    encrypted_message = data['encrypted_message']
    response = dkg_decrypt_message(start_node, encrypted_message)

    if response['message'] == 'Error.':
        return make_response(dumps(response), 400)
    return make_response(dumps(response), 201)

def dkg_decrypt_message(start_node, encrypted_message):
    '''
    Decrypts encrypted message using DKG
    '''

    # Split encrypted message into components
    encrypted_split_message = encrypted_message.split("/")

    # Decrypt each component
    decrypted_split_message = []
    for esm in encrypted_split_message:
        response = decrypt_command(start_node, esm.strip())
        parsed_response = dkg_decrypt_response(response)
        if parsed_response['message'] == "Error.":
            return parsed_response
        decrypted_split_message.append(parsed_response['data']['decrypted_message'])

    # Combine decrypted_split_message into one decrypted message
    decrypted_message = "".join(decrypted_split_message)

    response = {
        "message": "Success.",
        "data": {
            "decrypted_message": decrypted_message,
        }
    }

    return response

def decrypt_command(start_node, ct):
    '''
    Submits dkg decrypt to cli, returns response
    '''
    command = f'''
        dkgcli --config /tmp/node{start_node} dkg decrypt --encrypted {ct}
    '''
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    response = process.stdout.read().decode('utf-8')

    return response

def dkg_decrypt_response(response):
    '''
    Reads response from dkg decrypt command
    Parses response
    '''
    response_split = response.split(";")
    if len(response_split) != 3:
        return {
            "message": "Error.",
            "data": "DKG failed to decrypt."
        }

    if response_split[0] != 'DECRYPT':
        return {
            "message": "Error.",
            "data": "DKG failed to decrypt."
        }

    if response_split[0] == 'DECRYPT' and response_split[1] == "error":
        return {
            "message": "Error.",
            "data": response_split[2],
        }

    if response_split[0] == 'DECRYPT' and response_split[1] == "success":
        pt = response_split[2]
        return {
            "message": "Success.",
            "data": {
                "decrypted_message": bytes.fromhex(pt).decode('utf-8'),
            }
        }

###############################################################
#################### MASTER CREDENTIAL ########################
###############################################################

def dkg_issue_master_credential(user_id, start_node, num_nodes_mpc, data):
    '''
    Issues master credential
    '''
    fields = ['id']
    for field in fields:
        if not field in data:
            return make_response(
                dumps(
                    {
                        "message": "Issue Master Credential requires id_hash.",
                        "data": {}
                    }
                ),
                400
            )

    # Get id_hash from name
    user_name = data['id']

    # mpc hash
    k = random.getrandbits(128)
    num_rounds = 133
    id_hash = determine_mpc_hash(user_name, num_nodes_mpc, k, num_rounds)
    if id_hash == None:
        return make_response(
                dumps(
                    {
                        "message": "Error.",
                        "data": "Failed to determine mpc hash of user id",
                    }
                ),
                400
            )

    # Submit issue_master_credential command to identification system
    response = issue_master_credential_command(start_node, id_hash)

    # If error, return response
    if response == None:
        return make_response(
                dumps(
                    {
                        "message": "Error.",
                        "data": {}
                    }
                ),
                400
            )

    # Parse response
    response_split = response.split(":")
    master_credential = response_split[0]
    master_signatures = response_split[1:]

    # Store master credentials and signatures in db
    User.update_user_attribute("_id", user_id, "id_hash", id_hash)
    User.update_user_attribute("_id", user_id, "master_credential", master_credential)
    User.update_user_attribute("_id", user_id, "master_signatures", master_signatures)

    # If success, return response
    return make_response(
        dumps({
            "message": "Success.",
            "data": {
                "master_credential": master_credential,
                "master_signatures": master_signatures
        }}), 201)

def issue_master_credential_command(start_node, id_hash):
    command = f'dkgcli --config /tmp/node{start_node} dkg issueMasterCredential --idhash {id_hash}'
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    master_credential = process.stdout.read().decode('utf-8')
    time.sleep(1)

    return master_credential

#####################################################################
##################### ISSUE EVENT CREDENTIAL ########################
#####################################################################

def dkg_auth_event(user_id, num_nodes_dkg, start_node_dkg, data):
    '''
    Issues event credential
    '''
    # Get event_name from data
    fields = ['event_name']
    for field in fields:
        if not field in data:
            return make_response(
                dumps(
                    {
                        "message": "Issue Event Credential requires event_name.",
                        "data": {}
                    }
                ),
                400
            )
    event_name = data['event_name']

    is_user_authorised = dkg_auth_event_tx(user_id, event_name, num_nodes_dkg, start_node_dkg)

    # If error, return response
    if is_user_authorised['message'] == "Error.":
        return make_response(dumps(is_user_authorised), 400)
    
    # If success, return response
    return make_response(
        dumps(is_user_authorised), 201)


def dkg_auth_event_tx(user_id, event_name, num_nodes_dkg, start_node_dkg):
    '''
    Determines if user is authorized to submit an event contract transactions
    Verifies user event credentials
    '''
    # Get user
    user = User.find_user_by_attribute("_id", user_id)
    if user is None or (not (hasattr(user, 'id_hash') and hasattr(user, 'master_credential') and hasattr(user, 'master_signatures'))):
        return {
            "message": "Error",
            "data": "User does not have master credential"
        }
    
    # If user does not have event_credential and event_signatures, obtain event_credential and event_signatures
    id_hash = user.id_hash
    master_credential = user.master_credential
    master_signatures = user.master_signatures
    if (user.event_credential == "" or user.event_signatures == []):
        # If has master credential but not event credential, get event credential
        response = issue_event_credential_command(start_node_dkg, id_hash, event_name, master_credential, encode_signatures(master_signatures))
        event_credential, event_signatures = decode_credential(num_nodes_dkg, response)
        if event_credential is None:
            return {
                "message": "Error",
                "data": "User was not able to obtain event credential"
            }
    
        # Upload event credential to db
        User.update_user_attribute("_id", user_id, "event_credential", event_credential)
        User.update_user_attribute("_id", user_id, "event_signatures", event_signatures)

    # Verify event credential
    event_credential = user.event_credential
    event_signatures = user.event_signatures
    response = verify_event_credential(start_node_dkg, id_hash, event_name, event_credential, encode_signatures(event_signatures))

    # If verified, return true
    if response.strip() == "true":
        return {
            "message": "Success.",
            "data": ""
        }
    
    return {
        "message": "Error.",
        "data": "User was not able to verify event credential"
    }

def issue_event_credential_command(start_node, id_hash, event_name, master_credential, master_signatures):
    '''
    Submits issue_event_credential command to identification system
    '''
    command = f'dkgcli --config /tmp/node{start_node} dkg issueEventCredential --idhash {id_hash} --eventName {event_name} --masterCredential {master_credential} --masterSignatures {master_signatures}'
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    eventCredential = process.stdout.read().decode('utf-8')
    time.sleep(1)

    return eventCredential

#####################################################################
##################### VERIFY EVENT CREDENTIAL #######################
#####################################################################

def verify_event_credential(start_node, id_hash, event_name, event_credential, event_signatures):
    '''
    Submits verify_event_credential command to identification system, returns rsponse
    '''
    command = f'dkgcli --config /tmp/node{start_node} dkg verifyEventCredential --idhash {id_hash} --eventName {event_name} --eventCredential {event_credential} --eventSignatures {event_signatures}'
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    eventCredential = process.stdout.read().decode('utf-8')
    time.sleep(1)

    return eventCredential

#####################################################################
######################## HELPER FUNCTIONS ###########################
#####################################################################

def decode_credential(num_nodes_dkg, credential):
    '''
    Decodes master or event credential responses from dkg
    '''
    credential_split = credential.split(":")
    if len(credential_split) != num_nodes_dkg + 1:
        return None, None
    
    event_credential = credential_split[0]
    event_signatures = credential_split[1:]
    return event_credential, event_signatures

def encode_signatures(signatures):
    '''
    Encodes master and event signatures
    '''
    return ":".join(signatures)