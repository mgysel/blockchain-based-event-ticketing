from flask import Flask, request, make_response
from flask_cors import CORS
import jwt
from json import dumps, load
from functools import wraps
import sys
from flask.json import jsonify
from bson.objectid import ObjectId
import json

from routes.routes.event import event_create_event, event_buy_ticket, event_get_events, event_get_event, event_resell_ticket, event_rebuy_ticket, event_read_all, event_decrypt_execute_secondary, event_handle_secondary_transactions, event_use_ticket
from routes.routes.auth import auth_register, auth_login
from routes.routes.dkg import dkg_encrypt_message_server, dkg_decrypt_message_server, dkg_issue_master_credential, dkg_auth_event
from routes.routes.user import user_get_profile
from routes.routes.value import value_read_server, value_write_server, value_delete_server, value_list_server
from routes.objects.userObject import User

APP = Flask(__name__)
# Allows cross-origin AJAX, so React can talk to this API
CORS(APP)
APP.config['SECRET_KEY'] = 'your secret key'
APP.config['CORS_HEADERS'] = 'Content-Type'

global SERVER_PORT
global NUM_NODES_DELA 
global NUM_NODES_DKG
global NUM_NODES_MPC
global START_PORT_DELA
global START_NODE_DELA
global START_PORT_DKG
global START_NODE_DKG

#################################################
##################### AUTH ######################
#################################################

# decorator for verifying the JWT 
def token_required(f): 
    '''
    Decorator for verifying the JWT
    Use of JWT references:
    https://www.geeksforgeeks.org/using-jwt-for-user-authentication-in-flask/
    '''
    @wraps(f) 
    def decorated(*args, **kwargs): 
        token = None
        # jwt is passed in the request header
        if 'x-access-token' in request.headers:
            token = request.headers['x-access-token']
        # return 401 if token is not passed
        if not token:
            return jsonify({'message' : 'Token is missing.'}), 401

        try:
            # decoding the payload to fetch the stored details
            data = jwt.decode(token, APP.secret_key, algorithms="HS256")
            current_user = User.find_user_by_attribute("_id", ObjectId(data['id']))
        except Exception as e:
            return make_response(
                dumps(
                    {"message": f"Token is invalid: {e}"}
                ),
                401
            )
        # returns the current logged in users context to the routes
        return  f(current_user, *args, **kwargs)
   
    return decorated

@APP.route('/auth/register', methods=['POST'])
def register_user():
    '''
    Registers a user
    '''
    data = request.get_json()
    result = auth_register(data, APP.secret_key)
    return result

@APP.route('/auth/login', methods=['POST'])
def login_user():
    '''
    Logs in a user
    '''
    data = request.get_json()
    result = auth_login(data, APP.secret_key)
    return result

#################################################
##################### USEER #####################
#################################################

@APP.route('/user/profile', methods=['GET'])
@token_required
def user_profile(current_user):
    '''
    Gets user profile
    '''
    result = user_get_profile(current_user._id)
    return result

#################################################
##################### VALUE #####################
#################################################

@APP.route('/sc/value/write', methods=['POST'])
def sc_value_write():
    '''
    Writes a key-value pair to value contract
    Given key and value
    '''
    data = request.get_json()
    response = value_write_server(data, START_NODE_DELA)
    return response

@APP.route('/sc/value/read', methods=['GET'])
def sc_value_read():
    '''
    Reads a value from value contract
    Given key
    '''
    response = value_read_server(request.args, START_NODE_DELA)
    return response

@APP.route('/sc/value/delete', methods=['POST'])
def sc_value_delete():
    '''
    Deletes a key-value pair from value contract
    Given key
    '''
    data = request.get_json()
    response = value_delete_server(data, START_NODE_DELA)
    return response

@APP.route('/sc/value/list', methods=['GET'])
def sc_value_list():
    '''
    Returns a list of values from value smart contract
    '''
    response = value_list_server(START_NODE_DELA)
    return response

#################################################
##################### EVENT #####################
#################################################

@APP.route('/sc/event/get-events', methods=['GET'])
@token_required
def events_get(current_user):
    '''
    Gets all events from blockchain
    '''
    response = event_get_events()
    return response

@APP.route('/sc/event/get-event/<event_id>', methods=['GET'])
@token_required
def event_get(current_user, event_id):
    '''
    Gets event_id from db
    '''
    response = event_get_event(START_NODE_DELA, event_id)
    return response

@APP.route('/sc/event/create', methods=['POST'])
@token_required
def event_create(current_user):
    '''
    Creates an event
    Writes event to blockchain
    '''
    data = request.get_json()
    response = event_create_event(current_user._id, START_NODE_DELA, data)
    return response

@APP.route('/sc/event/buy', methods=['POST'])
@token_required
def event_buy(current_user):
    '''
    Buys a ticket to an event
    Writes buy tx to blockchain
    '''
    data = request.get_json()
    response = event_buy_ticket(current_user._id, START_NODE_DELA, NUM_NODES_DKG, START_NODE_DKG, data)
    return response

@APP.route('/sc/event/resell', methods=['POST'])
@token_required
def event_resell(current_user):
    '''
    Encrypts resell tx and writes to blockchain
    '''
    data = request.get_json()
    response = event_resell_ticket(current_user._id, data, START_NODE_DELA, START_NODE_DKG)
    return response

@APP.route('/sc/event/rebuy', methods=['POST'])
@token_required
def event_rebuy(current_user):
    '''
    Encrypts rebuy tx and writes to blockchain
    '''
    data = request.get_json()
    response = event_rebuy_ticket(current_user._id, data, START_NODE_DELA, START_NODE_DKG, NUM_NODES_DKG)
    return response

@APP.route('/sc/event/decrypt-execute-secondary', methods=['GET'])
@token_required
def event_decrypt_execute_secondary_tx(current_user):
    '''
    Shuffles, decrypts, and executes all secondary txs
    Event contract then stores these tx, but does not change ownership
    '''
    response = event_decrypt_execute_secondary(current_user._id, request.args, START_NODE_DELA, START_NODE_DKG)
    return response

@APP.route('/sc/event/transact-secondary', methods=['GET'])
@token_required
def event_transact_secondary(current_user):
    '''
    Executes secondary transactions
    Event contract transfers funds and changes ownership of secondary tx
    '''
    response = event_handle_secondary_transactions(current_user._id, request.args, START_NODE_DELA)
    return response

@APP.route('/sc/event/use-ticket', methods=['POST'])
@token_required
def event_use_tickets(current_user):
    '''
    Uses numTickets tickets
    '''
    data = request.get_json()
    response = event_use_ticket(current_user._id, data, START_NODE_DELA, NUM_NODES_DKG, START_NODE_DKG)
    return response

@APP.route('/sc/event/read', methods=['GET'])
@token_required
def event_read(current_user):
    '''
    Reads event smart contract
    '''
    response = event_read_all(current_user._id, START_NODE_DELA)
    return response

#################################################
###################### F3B ######################
#################################################

# Encrypt with F3B committee
@APP.route('/dkg/encrypt', methods=['POST'])
@token_required
def dkg_encrypt(current_user):
    '''
    Encrypts message using dkg committee
    '''
    data = request.get_json()
    response = dkg_encrypt_message_server(current_user._id, START_NODE_DKG, data)
    return response

# Encrypt with F3B committee
@APP.route('/dkg/decrypt', methods=['POST'])
@token_required
def dkg_decrypt(current_user):
    '''
    Decryptes encrypted message using dkg
    '''
    data = request.get_json()
    response = dkg_decrypt_message_server(current_user._id, START_NODE_DKG, data)
    return response

# Issues Master Credential with DKG committee
@APP.route('/dkg/issue-master-credential', methods=['POST'])
@token_required
def dkg_issue_mc(current_user):
    '''
    Issues master credential given id_hash
    '''
    data = request.get_json()
    response = dkg_issue_master_credential(current_user._id, START_NODE_DKG, NUM_NODES_MPC, data)
    return response

# Issues Event Credential with DKG committee
@APP.route('/dkg/auth-event-tx', methods=['POST'])
@token_required
def dkg_auth_eventtx(current_user):
    '''
    Authorizes user based on their event credential
    '''
    data = request.get_json()
    response = dkg_auth_event(current_user._id, NUM_NODES_DKG, START_NODE_DKG, data)
    return response

#################################################
#################### SERVER #####################
#################################################

if __name__ == "__main__":
    try:
        # Initialize variables
        with open('setup_info.json', 'r') as f:
            start_info = json.load(f)
            print("Start info: ", start_info)
            if "server_port" in start_info and "num_nodes_dela" in start_info and "num_nodes_dkg" in start_info and "num_nodes_mpc" in start_info and "start_node_dela" in start_info and "start_port_dela" in start_info and "start_node_dkg" in start_info and "start_port_dkg" in start_info:
                SERVER_PORT = start_info['server_port']
                NUM_NODES_DELA = start_info['num_nodes_dela']
                NUM_NODES_DKG = start_info['num_nodes_dkg']
                NUM_NODES_MPC = start_info['num_nodes_mpc']
                START_NODE_DELA = start_info['start_node_dela']
                START_PORT_DELA = start_info['start_port_dela']
                START_NODE_DKG = start_info['start_node_dkg']
                START_PORT_DKG = start_info['start_port_dkg']
                APP.run(port=(int(sys.argv[1]) if len(sys.argv) == 2 else SERVER_PORT), debug=True)
    except Exception as e:
        print("Error: ", e)
        print("setup_info.json must contain server_port, num_nodes_dela, num_nodes_dkg, num_nodes_mpc, start_node_dela, start_port_dela, start_node_dkg, start_port_dkg")