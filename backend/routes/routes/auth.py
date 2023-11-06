import sys
sys.path.append("../..")
from flask import make_response
from json import dumps
from werkzeug.security import generate_password_hash, check_password_hash
import jwt
from datetime import datetime, timedelta 
from cryptography.hazmat.primitives.asymmetric import ec
from cryptography.hazmat.primitives import serialization

from routes.objects.userObject import User


########## MAIN FUNCTIONS ##########

def auth_register(data, secret_key):
    '''
    Inputs email, password, first name, and last name
    Attempts to create new user
    Errors from invalid email, email taken, password < 6 characters,
    first or last name being ouside of 1 to 50 range
    '''
    fields = ['email', 'password', 'confirm_password']
    for field in fields:
        if not field in data:
            return make_response(
                dumps(
                    {"message": "User object is invalid."}
                ), 
                400
            ) 

    email = data['email']
    password = data['password']
    confirm_password = data['confirm_password']
    id_hash = ''
    master_credential = ''
    master_signatures = []
    event_credential = ''
    event_signatures = []
    # Check - valid email
    if not User.valid_email(email):
        return make_response(
            dumps(
                {"message": "Invalid Email Address."}
            ), 
            400
        ) 

    # Check - unused email
    if not User.unused_email(email):
        return make_response(
            dumps(
                {"message": "Email Address already taken."}
            ), 
            400
        ) 

    # Check - valid password
    if not User.valid_password(password):
        return make_response(
            dumps(
                {"message": "Password must be longer than 6 characters."}
            ), 
            400
        ) 
    if not User.valid_matching_passwords(password, confirm_password):
        return make_response(
            dumps(
                {"message": "Passwords must match."}
            ), 
            400
        ) 
    
    # Get a public/private key pair
    skObject = ec.generate_private_key(ec.SECP384R1())
    pkObject = skObject.public_key()
    sk = str(skObject.private_numbers().private_value)
    pk = pkObject.public_bytes(serialization.Encoding.X962, serialization.PublicFormat.UncompressedPoint).hex()

    # Create user object, add to database
    user = User(None, email, generate_password_hash(password), sk, pk, id_hash, master_credential, master_signatures, event_credential, event_signatures)
    User.insert_one(user)

    # Log user in once registered
    data = {
        'email': email,
        'password': password
    }
    return auth_login(data, secret_key)

def auth_login(data, secret_key):
    '''
    Inputs user email/password
    If valid email/password, logs in a user and returns JWTtoken
    Use of JWT references:
    https://www.geeksforgeeks.org/using-jwt-for-user-authentication-in-flask/
    '''
    fields = ['email', 'password']
    for field in fields:
        if not field in data:
            return make_response(
                dumps(
                    {"message": "User object is invalid."}
                ), 
                400
            ) 

    email = data['email']
    password = data['password']

    if not User.valid_email(email) or not User.valid_password(password): 
        # returns 401 if email/password not valid
        return make_response(
            dumps(
                {"message": "Invalid email or password."}
            ), 
            401
        ) 
        
    # Obtain user from database
    user = User.find_user_by_attribute("email", email)
   
    if not user: 
        # returns 401 if user does not exist 
        return make_response(
            dumps(
                {"message": "User does not exist."}
            ), 
            401
        ) 

    if check_password_hash(user.password, password): 
        
        # generates the JWT Token 
        token = jwt.encode({ 
            'id': str(user._id),
            # Token valid for 30 days
            'exp' : datetime.utcnow() + timedelta(days = 30) 
        }, secret_key) 

        return make_response(
            dumps(
                {
                    "token": token,
                    "PK": user.pk,
                    "message": "Success."
                }
            ), 
            201
        ) 
    
    # returns 403 if password is wrong 
    return make_response(
        dumps(
            {"message": "Incorrect email or password."}
        ), 
        403
    ) 
    

'''
To logout, tokens should be removed from the client side cookie
https://stackoverflow.com/questions/21978658/invalidating-json-web-tokens
def auth_logout(token, secret_key):
    @token_required(secret_key)
'''