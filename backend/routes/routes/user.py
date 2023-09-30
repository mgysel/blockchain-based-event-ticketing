import sys
sys.path.append("../..")
from flask import make_response
from json import dumps
from werkzeug.security import generate_password_hash, check_password_hash
import jwt
from datetime import datetime, timedelta 
from functools import wraps
import subprocess
import os
from cryptography.hazmat.primitives.asymmetric import ec
from cryptography.hazmat.primitives import hashes, serialization

from routes.objects.userObject import User

def user_get_profile(user_id):
    '''
    Gets user profile from databaase
    '''
        
    # Obtain user from database
    user = User.find_user_by_attribute("_id", user_id)
    print("User: ", user)
   
    if not user: 
        # returns 401 if user does not exist 
        return make_response(
            dumps(
                {
                    "message": "Error.",
                    "data": "User does not exist.",
                }
            ), 
            401
        ) 

    return make_response(
        dumps(
            {
                "message": "Success.",
                "data": {
                    "master_credential": user.master_credential,
                    "id_hash": user.id_hash,
                    "pk": user.pk,
                },
            }
        ), 
        201
    ) 