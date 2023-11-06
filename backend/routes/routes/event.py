import os

from flask import make_response
from json import dumps, loads
from bson.objectid import ObjectId
import subprocess
import time 
import random 

from routes.objects.eventObject import Event
from routes.objects.userObject import User
from routes.routes.dkg import dkg_auth_event_tx, dkg_encrypt_message, dkg_decrypt_message
from routes.routes.value import value_write, value_list, value_delete

NUM_REBUY_TX = 0

###############################################################
########################## GET EVENTS #########################
###############################################################

def event_get_events():
    '''
    Returns all events from the database
    '''
    events = Event.get_all_events()

    if isinstance(events, list):
        if len(events) > 0:
            return make_response(
                dumps(
                    {
                        "message": "Success.",
                        "data": Event.many_to_json_str(events),
                    }
                ),
                201
            )
        else:
            return make_response(
                dumps(
                    {
                        "message": "Success.",
                        "data": [],
                    }
                ),
                201
            )

    return make_response(
        dumps(
            {
                "message": "Error.",
                "data": "Events do not exist in the database.",
            }
        ),
        400
    )

###############################################################
########################## GET EVENT ##########################
###############################################################

def event_get_event(start_node, event_id):
    '''
    Retrieves event of event_id
    '''
    try:
        # Find event by event_id
        oid = ObjectId(event_id)
        event = Event.find_event_by_attribute("_id", oid)

        if event:
            event_json = Event.to_json(event)

            return {
                "message": "Success.",
                "data": event_json,
            }, 200

        return {f"message": "Could not find event {event_id}."}, 404

    except Exception as e:
        return {f"message": "Could not find  {event_id}: {e}."}, 404

########################################################################################################################
################################################# EVENT CONTRACT #######################################################
########################################################################################################################

######################################################
##################### INIT ###########################
######################################################

def event_create_event(user_id, start_node, data):
    '''
    Creates an event on the dela blockchain 
    given name, num_tickets, price, max_resale_price, resale_royalty
    '''
    fields = ['name','num_tickets', 'price', 'max_resale_price', 'resale_royalty']
    for field in fields:
        if not field in data:
            return make_response(
                dumps(
                    {
                        "message": "Event requires name, num_tickets, price, max_resale_price, resale_royalty.",
                        "data": {}
                    }
                ),
                400
            )

    name = data['name']
    num_tickets = data['num_tickets']
    price = data['price']
    max_resale_price = data['max_resale_price']
    resale_royalty = data['resale_royalty']

    # Get public key for user
    pk = get_user_pk(user_id)
    if pk is None:
        return make_response(
            dumps(
                {
                    "message": "Error.",
                    "data": "Could not obtain user public key from database.",
                }
            ),
            400
        )

    # Submit event to blockchain
    response = event_contract_init(start_node, pk, name, num_tickets, price, max_resale_price, resale_royalty)
    if response is None:
        return make_response(
            dumps(
                {
                    "message": "Error.",
                    "data": "Error reading results from blockchain.",
                }
            ),
            400
        )

    # If error, return response
    if response['message'] == "Error.":
        return make_response(dumps(response), 400) 

    # If success, add to MongoDB
    # Make sure all fields are present
    fields = ['owner', 'name','num_tickets', 'price', 'max_resale_price', 'resale_royalty']
    for field in fields:
        if not field in response['data']:
            return make_response(
                dumps(
                    {
                        "message": "Event requires owner, name, num_tickets, price, max_resale_price, resale_royalty.",
                        "data": {}
                    }
                ),
                400
            )

    # Add to MongoDB
    owner = response['data']['owner']
    name = response['data']['name']
    num_tickets = response['data']['num_tickets']
    price = response['data']['price']
    max_resale_price = response['data']['max_resale_price']
    resale_royalty = response['data']['resale_royalty']
    n_rebuy_tx = 0 
    event = Event(None, owner, name, num_tickets, price, max_resale_price, resale_royalty, n_rebuy_tx)
    Event.insert_one(event)

    return make_response(dumps(response), 201)

def event_contract_init(start_node, initPK, initName, initNumTickets, initPrice, initMaxResalePrice, initResaleRoyalty):
    '''
    Sends tx to blockchain to init event smart contract
    Returns response from blockchain nodes
    '''
    event_contract_init_command(start_node, initPK, initName, initNumTickets, initPrice, initMaxResalePrice, initResaleRoyalty)
    time.sleep(1)
    response = event_init_text()
    return response

def event_contract_init_command(start_node, initPK, initName, initNumTickets, initPrice, initMaxResalePrice, initResaleRoyalty):
    '''
    Sends tx to blockchain to init event smart contract
    '''
    pk_path = os.getcwd() + '/dela/private.key'
    command = f'''
        memcoin\
        --config /tmp/node{start_node} pool add\
        --key {pk_path}\
        --args go.dedis.ch/dela.ContractArg --args go.dedis.ch/dela.Event\
        --args value:initPK --args {initPK}\
        --args value:initName --args {initName}\
        --args value:initNumTickets --args {initNumTickets}\
        --args value:initPrice --args {initPrice}\
        --args value:initMaxResalePrice --args {initMaxResalePrice}\
        --args value:initResaleRoyalty --args {initResaleRoyalty}\
        --args value:command --args INIT 
    '''
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    return process_output

def event_init_text():
    '''
    Returns init command response from blockchain nodes
    '''
    readlines = []
    fn = os.getcwd() + '/dela/outputs/dela_outputs.txt'
    with open(fn, 'r') as f:
        for line in f:
            if "//EVENTCONTRACT_INITOUTPUT" in line:
                readlines.append(line)

    readlines.reverse()
    if (len(readlines) > 0):
        line = readlines[0].split("//")[1]
        split_line = line.split(";")

        # Success
        if split_line[1] == "success" and len(split_line) == 9:
            tx_count = split_line[2]
            owner = split_line[3]
            name = split_line[4]
            num_tickets = split_line[5]
            price = split_line[6]
            max_resale_price = split_line[7]
            resale_royalty = split_line[8]
            if (int(tx_count) == 1):
                return {
                    "message": "Success.",
                    "data": {
                        "tx_count": tx_count,
                        "owner": owner,
                        "name": name,
                        "num_tickets": num_tickets,
                        "price": price,
                        "max_resale_price": max_resale_price,
                        "resale_royalty": resale_royalty
                    }
                }

        # Error
        elif split_line[2] == "error" and len(split_line) == 4:
            tx_count = split_line[3]
            message = split_line[4]
            return {
                "message": "Error.",
                "data": message,
            }

    return None

######################################################
###################### BUY ###########################
######################################################

def event_buy_ticket(user_id, start_node_dela, num_nodes_dkg, start_node_dkg, data):
    '''
    Buys a ticket to an event on the dela blockchain 
    given price
    '''
    fields = ['price', 'num_tickets', 'event_name']
    for field in fields:
        if not field in data:
            return make_response(
                dumps(
                    {
                        "message": "Buying ticket requires event_name, num_tickets, and price.",
                        "data": {}
                    }
                ),
                400
            )

    price = data['price']
    num_tickets = data['num_tickets']
    event_name = data['event_name']

    # Get public key for user
    pk = get_user_pk(user_id)
    if pk is None:
        return make_response(
            dumps(
                {
                    "message": "Error.",
                    "data": "Could not obtain user public key from database.",
                }
            ),
            400
        )

    # Verify user event credential 
    is_user_authorized = dkg_auth_event_tx(user_id, event_name, num_nodes_dkg, start_node_dkg)
    if not is_user_authorized:
        return make_response(
            dumps(
                {
                    "message": "Error.",
                    "data": "Could not verify user event credential.",
                }
            ),
            400
        )
    
    # Get user event credential
    event_credential = get_user_event_credential(user_id)
    if event_credential is None:
        return make_response(
            dumps(
                {
                    "message": "Error.",
                    "data": "Could not obtain user event credential from database.",
                }
            ),
            400
        )

    # Submit buy ticket to blockchain
    response = event_contract_buy(start_node_dela, pk, num_tickets, price, event_credential)

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

    if response['message'] == "Error.":
        return make_response(dumps(response), 400)

    # If success, return response
    return make_response(
        dumps(response),
        201
    )

def event_contract_buy(start_node, buy_pk, buy_num_tickets, buy_payment, event_credential):
    '''
    Sends tx to blockchain to buy a ticket from event smart contract
    Returns response from blockchain nodes
    '''
    event_contract_buy_command(start_node, buy_pk, buy_num_tickets, buy_payment, event_credential)
    time.sleep(1)
    response = event_buy_text()
    return response

def event_contract_buy_command(start_node, buy_pk, buy_num_tickets, buy_payment, event_credential):
    '''
    Sends tx to blockchain to buy a ticket
    '''

    pk_path = os.getcwd() + '/dela/private.key'
    command = f'''
        memcoin\
        --config /tmp/node{start_node} pool add\
        --key {pk_path}\
        --args go.dedis.ch/dela.ContractArg --args go.dedis.ch/dela.Event\
        --args value:buyPK --args {buy_pk}\
        --args value:buyNumTickets --args {buy_num_tickets}\
        --args value:buyPayment --args {buy_payment}\
        --args value:buyEventCredential --args {event_credential}\
        --args value:command --args BUY
    '''
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')

def event_buy_text():
    '''
    Returns response from blockchain of a buy command 
    '''
    readlines = []
    fn = f'{os.getcwd()}/dela/outputs/dela_outputs.txt'
    with open(fn, 'r') as f:
        for line in f:
            if "//EVENTCONTRACT_BUYOUTPUT" in line:
                readlines.append(line)

    readlines.reverse()
    if (len(readlines) > 0):
        line = readlines[0].split("//")[1]
        split_line = line.split(";")

        # Success
        if split_line[1] == "success" and len(split_line) == 6:
            tx_count = split_line[2]
            event_name = split_line[3]
            owner = split_line[4]
            payment = split_line[5]
            if tx_count.isnumeric():
                return {
                    "message": "Success.",
                    "data": {
                        "tx_count": int(tx_count),
                        "event_name": event_name,
                        "owner": owner,
                        "payment": payment,
                    }
                }
        
        # Error
        elif split_line[2] == "error" and len(split_line) == 4:
            message = split_line[3]
            return {
                "message": "Error.",
                "data": message,
            }

    return None

######################################################
#################### RESELL ##########################
######################################################

def event_resell_ticket(user_id, data, start_node_dela, start_node_dkg):
    '''
    Reselling a ticket to an event on the dela blockchain 
    given event_name, price, and num_tickets
    '''
    fields = ['event_name', 'price', 'num_tickets']
    for field in fields:
        if not field in data:
            return make_response(
                dumps(
                    {
                        "message": "Reselling ticket requires event_name, price, num_tickets.",
                        "data": {}
                    }
                ),
                400
            )

    event_name = data['event_name']
    price = data['price']
    num_tickets = data['num_tickets']

    # Get user pk
    pk = get_user_pk(user_id)
    if pk is None:
        return make_response(
            dumps(
                {
                    "message": "Error.",
                    "data": "Could not obtain user public key from database.",
                }
            ),
            400
        )

    # Get user event credential
    event_credential = get_user_event_credential(user_id)
    if event_credential is None:
        return make_response(
            dumps(
                {
                    "message": "Error.",
                    "data": "Could not obtain user event credential from database.",
                }
            ),
            400
        )

    # Create resell transaction
    resell_tx = 'resell;' + str(event_name) + ';' + str(pk) + ';' + str(num_tickets) + ';' + str(price) + ";" + str(event_credential)

    # Dkg encrypt resell transaction
    response = dkg_encrypt_message(start_node_dkg, resell_tx)
    if response['message'] == 'Error.':
        return make_response(dumps(response), 400)
    encrypted_resell_tx = response['data']['encrypted_message']

    # Write encrypted resell transaction to Dela
    secondary_tx_id = random.randint(0, 1000000000000)
    response = value_write(start_node_dela, f"secondary_tx_{secondary_tx_id}", str(encrypted_resell_tx))
    return make_response(
        dumps(
            {
                "message": "Success.",
                "data": {
                    "encrypted_resell_tx": encrypted_resell_tx,
                }
            }
        ),
        201
    )

######################################################
##################### REBUY ##########################
######################################################

def event_rebuy_ticket(user_id, data, start_node_dela, start_node_dkg, num_nodes_dkg):
    '''
    Rebuying a ticket to an event on the dela blockchain 
    given price
    '''
    fields = ['event_name', 'price', 'num_tickets']
    for field in fields:
        if not field in data:
            return make_response(
                dumps(
                    {
                        "message": "Rebuying ticket requires price.",
                        "data": {}
                    }
                ),
                400
            )

    event_name = data['event_name']
    price = data['price']
    num_tickets = data['num_tickets']

    # Get user pk
    pk = get_user_pk(user_id)
    if pk is None:
        return make_response(
            dumps(
                {
                    "message": "Error.",
                    "data": "Could not obtain user public key from database.",
                }
            ),
            400
        )

    # Verify user event credential 
    is_user_authorized = dkg_auth_event_tx(user_id, event_name, num_nodes_dkg, start_node_dkg)
    if not is_user_authorized:
        return make_response(
            dumps(
                {
                    "message": "Error.",
                    "data": "Could not verify user event credential.",
                }
            ),
            400
        )

    # Get user event credential
    event_credential = get_user_event_credential(user_id)
    if event_credential is None:
        return make_response(
            dumps(
                {
                    "message": "Error.",
                    "data": "Could not obtain user event credential from database.",
                }
            ),
            400
        )

    # Create rebuy transaction
    rebuy_tx = 'rebuy;' + str(event_name) + ';' + str(pk) + ';' + str(num_tickets) + ';' + str(price) + ";" + str(event_credential)

    # Dkg encrypt rebuy transaction
    response = dkg_encrypt_message(start_node_dkg, rebuy_tx)
    if response['message'] == 'Error.':
        return make_response(dumps(response), 400)
    encrypted_rebuy_tx = response['data']['encrypted_message']

    # Write encrypted rebuy transaction to Dela
    secondary_tx_id = random.randint(0, 1000000000000)
    response = value_write(start_node_dela, f"secondary_tx_{secondary_tx_id}", str(encrypted_rebuy_tx))

    return make_response(
        dumps(
            {
                "message": "Success.",
                "data": {
                    "encrypted_rebuy_tx": encrypted_rebuy_tx,
                }
            }
        ),
        201
    )

###################################################################
################## DECRYPT/EXECUTE SECONDARY ######################
###################################################################

def event_decrypt_execute_secondary(user_id, args, start_node_dela, start_node_dkg):
    '''
    Decrypts and executes secondary market transactions
    '''
    event_name = args.get('name', None)
    if event_name is None:
        return make_response(
            dumps(
                {
                    "message": "Handle Rebuy Ticket requires event_name.",
                    "data": {}
                }
            ),
            400
        )

    # Get list of secondary transactions
    response = value_list(start_node_dela)
    if response['message'] == "Error.":
        return make_response(dumps(response), 400)
    list_values = response['data']['pairs']

    # Get encrypted secondary txs
    encrypted_secondary_txs_keys = []
    encrypted_secondary_txs = []
    for lv in list_values:
        if lv['key'].startswith("secondary_tx_"):
            encrypted_secondary_txs.append(lv['value'])
            encrypted_secondary_txs_keys.append(lv['key'])

    # Shuffle encrypted secondary tx
    random.shuffle(encrypted_secondary_txs)

    # Decrypt each secondary tx 
    secondary_txs = []
    for est in encrypted_secondary_txs:
        response = dkg_decrypt_message(start_node_dkg, est)
        if response['message'] == "Error.":
            return make_response(dumps(response), 400)
        secondary_tx_split = response['data']['decrypted_message'].split(";")
        if len(secondary_tx_split) != 6:
            return make_response(
                dumps(
                    {
                        "message": "Error.",
                        "data": "Decrypted Rebuy tx is not in correct format.",
                    }
                ),
                400
            )
        tx_type = secondary_tx_split[0]
        event_name = secondary_tx_split[1]
        pk = secondary_tx_split[2]
        num_tickets = secondary_tx_split[3]
        price = secondary_tx_split[4]
        event_credential = secondary_tx_split[5]
        secondary_txs.append({
            "tx_type": tx_type,
            "event_name": event_name,
            "pk": pk,
            "num_tickets": num_tickets,
            "price": price,
            "event_credential": event_credential
        })

    # Execute each secondary tx 
    for st in secondary_txs:
        if st['tx_type'] == "rebuy":
            response = event_contract_rebuy(start_node_dela, st['pk'], st['num_tickets'], st['price'], st['event_credential'])
        elif st['tx_type'] == "resell":
            response = event_contract_resell(start_node_dela, st['pk'], st['num_tickets'], st['price'], st['event_credential'])

    # Delete secondary txs
    for k in encrypted_secondary_txs_keys:
        response = value_delete(k, start_node_dela)
    
    return make_response(dumps(
        {
            "message": "Success.",
            "data": {}
        }
    ))

def event_contract_rebuy(start_node_dela, rebuy_pk, rebuy_num_tickets, rebuy_price, event_credential):
    '''
    Sends tx to blockchain to buy a ticket from event smart contract
    Returns response from blockchain nodes
    '''
    event_contract_rebuy_command(start_node_dela, rebuy_pk, rebuy_num_tickets, rebuy_price, event_credential)
    response = event_rebuy_text()
    return response

def event_contract_rebuy_command(start_node_dela, rebuy_pk, rebuy_num_tickets, rebuy_price, event_credential):
    '''
    Sends tx to blockchain to rebuy a ticket to event smart contract
    Returns response from blockchain nodes
    '''

    pk_path = os.getcwd() + '/dela/private.key'
    command = f'''
        memcoin\
        --config /tmp/node{start_node_dela} pool add\
        --key {pk_path}\
        --args go.dedis.ch/dela.ContractArg --args go.dedis.ch/dela.Event\
        --args value:rebuyPK --args {rebuy_pk}\
        --args value:rebuyNumTickets --args {rebuy_num_tickets}\
        --args value:rebuyPrice --args {rebuy_price}\
        --args value:rebuyEventCredential --args {event_credential}\
        --args value:command --args REBUY
    '''

    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')
    time.sleep(1)

def event_rebuy_text():
    '''
    Returns response from blockchain of a rebuy command 
    '''
    readlines = []
    fn = f'{os.getcwd()}/dela/outputs/dela_outputs.txt'
    with open(fn, 'r') as f:
        for line in f:
            if "//EVENTCONTRACT_REBUYOUTPUT" in line:
                readlines.append(line)

    readlines.reverse()
    if (len(readlines) > 0):
        line = readlines[0].split("//")[1]
        split_line = line.split(";")

        # Success
        if split_line[1] == "success" and len(split_line) == 6:
            tx_count = split_line[2]
            pk = split_line[3]
            event_name = split_line[4]
            num_tickets = split_line[5]
            return {
                "message": "Success.",
                "data": {
                    "event_name": event_name,
                    "num_tickets": num_tickets,
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
        "data": "Failed to read rebuy tx output.",
    }

def event_contract_resell(start_node, pk, num_tickets, price, event_credential):
    '''
    Sends tx to blockchain to resell a ticket from event smart contract
    Returns response from blockchain nodes
    '''
    event_contract_resell_command(start_node, pk, num_tickets, price)
    time.sleep(1)
    response = event_resell_text()
    return response

def event_contract_resell_command(start_node, pk, num_tickets, price):
    '''
    Sends tx to blockchain to resell a ticket to event smart contract
    Returns response from blockchain nodes
    '''

    pk_path = os.getcwd() + '/dela/private.key'
    command = f'''
        memcoin\
        --config /tmp/node{start_node} pool add\
        --key {pk_path}\
        --args go.dedis.ch/dela.ContractArg --args go.dedis.ch/dela.Event\
        --args value:resellPK --args {pk}\
        --args value:resellNumTickets --args {num_tickets}\
        --args value:resellPrice --args {price}\
        --args value:command --args RESELL
    '''
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')

def event_resell_text():
    '''
    Returns response from blockchain of a resell command 
    '''
    readlines = []
    fn = f'{os.getcwd()}/dela/outputs/dela_outputs.txt'
    with open(fn, 'r') as f:
        for line in f:
            if "//EVENTCONTRACT_RESELL" in line:
                readlines.append(line)

    readlines.reverse()
    if (len(readlines) > 0):
        line = readlines[0].split("//")[1]
        split_line = line.split(";")

        # Success
        if split_line[1] == "success" and len(split_line) == 6:
            event_name = split_line[3]
            num_tickets = split_line[4]
            price = split_line[5]
            return {
                "message": "Success.",
                "data": {
                    "event_name": event_name,
                    "num_tickets": num_tickets,
                    "price": price,
                }
            }
        
        # Error
        elif split_line[2] == "error" and len(split_line) == 4:
            message = split_line[3]
            return {
                "message": "Error.",
                "data": message,
            }
        
    # Clear/Close file
    # open(fn, 'w').close()

    return None

###################################################################
####################### HANDLE SECONDARY ##########################
###################################################################

def event_handle_secondary_transactions(user_id, args, start_node_dela):
    '''
    Handles secondary market transactions once they are decrypted and executed
    '''
    event_name = args.get('name', None)
    if event_name is None:
        return make_response(
            dumps(
                {
                    "message": "Handle Secondary Transactions requires event_name.",
                    "data": {}
                }
            ),
            400
        )

    # Call handle Resales
    user_pk = get_user_pk(user_id)
    response = event_contract_handle_resales(start_node_dela, user_pk)
    
    return make_response(dumps(
        {
            "message": "Success.",
            "data": {}
        }
    ))

def event_contract_handle_resales(start_node_dela, handle_resales_pk):
    '''
    Sends tx to blockchain to handle resales
    Returns response from blockchain nodes
    '''
    event_contract_handle_resales_command(start_node_dela, handle_resales_pk)
    response = event_handle_resales_text()
    return response

def event_contract_handle_resales_command(start_node_dela, handle_resales_pk):
    '''
    Sends tx to blockchain to handle resales
    Returns response from blockchain nodes
    '''

    pk_path = os.getcwd() + '/dela/private.key'
    command = f'''
        memcoin\
        --config /tmp/node{start_node_dela} pool add\
        --key {pk_path}\
        --args go.dedis.ch/dela.ContractArg --args go.dedis.ch/dela.Event\
        --args value:handleResalesPK --args {handle_resales_pk}\
        --args value:command --args HANDLERESALES
    '''

    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')
    print(f'Process output: {process_output}')

def event_handle_resales_text():
    '''
    Returns response from blockchain of a handleResales command 
    '''
    readlines = []
    fn = f'{os.getcwd()}/dela/outputs/dela_outputs.txt'
    with open(fn, 'r') as f:
        for line in f:
            if "//EVENTCONTRACT_HANDLERESALESOUTPUT" in line:
                readlines.append(line)

    readlines.reverse()
    if (len(readlines) > 0):
        line = readlines[0].split("//")[1]
        split_line = line.split(";")

        # Success
        if split_line[1] == "success" and len(split_line) == 4:
            tx_count = split_line[2]
            event_name = split_line[3]
            return {
                "message": "Success.",
                "data": {
                    "event_name": event_name,
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
        "data": "Failed to read rebuy tx output.",
    }

###################################################################
########################## USE TICKET #############################
###################################################################

def event_use_ticket(user_id, data, start_node_dela, num_nodes_dkg, start_node_dkg):
    '''
    Uses user tickets
    given num_tickets and event_name
    '''
    fields = ['num_tickets', 'event_name', 'id']
    for field in fields:
        if not field in data:
            return make_response(
                dumps(
                    {
                        "message": "Using ticket requires event_name, num_tickets",
                        "data": {}
                    }
                ),
                400
            )

    num_tickets = data['num_tickets']
    event_name = data['event_name']
    id = data['id']

    # Get public key for user
    pk = get_user_pk(user_id)
    if pk is None:
        return make_response(
            dumps(
                {
                    "message": "Error.",
                    "data": "Could not obtain user public key from database.",
                }
            ),
            400
        )
    
    # Verify user event credential 
    is_user_authorized = dkg_auth_event_tx(user_id, event_name, num_nodes_dkg, start_node_dkg)
    if not is_user_authorized:
        return make_response(
            dumps(
                {
                    "message": "Error.",
                    "data": "Could not verify user event credential.",
                }
            ),
            400
        )
    
    # Get event credential for user
    event_credential = get_user_event_credential(user_id)
    if event_credential is None:
        return make_response(
            dumps(
                {
                    "message": "Error.",
                    "data": "Could not obtain event credential from database.",
                }
            ),
            400
        )

    # Submit use ticket to blockchain
    event_contract_use_ticket(start_node_dela, pk, num_tickets, event_credential)

    # If success, return response
    return make_response(
        dumps({
            "message": "Success.",
            "data": {}
        }),
        201
    )

def event_contract_use_ticket(start_node, pk, num_tickets, event_credential):
    '''
    Sends tx to blockchain to use a ticket from event smart contract
    Returns response from blockchain nodes
    '''
    event_contract_use_ticket_command(start_node, pk, num_tickets, event_credential)

def event_contract_use_ticket_command(start_node, pk, num_tickets, event_credential):
    '''
    Sends tx to blockchain to use tickets
    '''
    pk_path = os.getcwd() + '/dela/private.key'
    command = f'''
        memcoin\
        --config /tmp/node{start_node} pool add\
        --key {pk_path}\
        --args go.dedis.ch/dela.ContractArg --args go.dedis.ch/dela.Event\
        --args value:useTicketPK --args {pk}\
        --args value:useTicketNumTickets --args {num_tickets}\
        --args value:useTicketEventCredential --args {event_credential}\
        --args value:command --args USETICKET
    '''
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')

###############################################################
########################### READ ALL ##########################
###############################################################

def event_read_all(user_id, start_node):
    '''
    Reads event smart contract data
    '''

    # Submit read all to blockchain
    response = event_contract_read_all(start_node)

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

    if response['message'] == "Error.":
        return make_response(dumps(response), 400)

    # If success, return response
    return make_response(
        dumps(response),
        201
    )

def event_contract_read_all(start_node):
    '''
    Sends tx to blockchain to read event smart contract
    Returns response from blockchain nodes
    '''
    event_contract_read_all_command(start_node)
    time.sleep(1)
    response = event_read_all_text()
    return response

def event_contract_read_all_command(start_node):
    '''
    Sends tx to blockchain to read event smart contract
    '''

    pk_path = os.getcwd() + '/dela/private.key'
    command = f'''
        memcoin\
        --config /tmp/node{start_node} pool add\
        --key {pk_path}\
        --args go.dedis.ch/dela.ContractArg --args go.dedis.ch/dela.Event\
        --args value:command --args READEVENTCONTRACT
    '''
    process = subprocess.Popen(command, shell=True, stdout=subprocess.PIPE)
    process.wait()
    process_output = process.stdout.read().decode('utf-8')

def event_read_all_text():
    '''
    Returns response from blockchain of a readevent command 
    '''
    readlines = []
    fn = f'{os.getcwd()}/dela/outputs/dela_outputs.txt'
    with open(fn, 'r') as f:
        for line in f:
            if "//EVENTCONTRACT_READEVENTOUTPUT" in line:
                readlines.append(line)

    readlines.reverse()
    if (len(readlines) > 0):
        line = readlines[0].split("//")[1]
        split_line = line.split(";")

        # Success
        if split_line[1] == "success" and len(split_line) == 5:
            tx_count = split_line[2]
            event_name = split_line[3]
            contract_data = split_line[4]
            return {
                "message": "Success.",
                "data": {
                    "tx_count": tx_count,
                    "event_name": event_name,
                    "contract_data": loads(contract_data),
                }
            }
        
        # Error
        elif split_line[2] == "error" and len(split_line) == 4:
            message = split_line[3]
            return {
                "message": "Error.",
                "data": message,
            }

    return None

###############################################################
####################### HELPER FUNCTIONS ######################
###############################################################

def get_user_pk(user_id):
    '''
    Gets user secret key given user_id
    '''
    user = User.find_user_by_attribute("_id", user_id)
    if hasattr(user, 'pk'):
        return user.pk
    return None

def get_user_event_credential(user_id):
    '''
    Gets user event credential given user_id
    '''
    user = User.find_user_by_attribute("_id", user_id)
    if hasattr(user, 'event_credential'):
        return user.event_credential
    return None