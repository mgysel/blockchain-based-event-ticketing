import sys
sys.path.append("../..")

import pymongo

# If we use the Flask Configuration
'''
import sys
sys.path
sys.path.append(".")
sys.path.append("..")
sys.path.append("./flask-app")
#from server import mongo
'''

from werkzeug.security import generate_password_hash
from routes.objects.MongoWrapper import MongoWrapper
from bson import ObjectId
import re

class User:
    '''
    User class that contains basic user info/methods
    '''
    def __init__(self, _id, email, password, sk, pk, id_hash, master_credential, master_signatures, event_credential, event_signatures):
        self._id = _id
        self.email = email
        self.password = password
        self.sk = sk
        self.pk = pk
        self.id_hash = id_hash
        self.master_credential = master_credential
        self.master_signatures = master_signatures
        self.event_credential = event_credential
        self.event_signatures = event_signatures

    def from_json(user_json):
        '''
        User json object to User Object
        '''
        if user_json != None:
            properties = ['email', 'password', 'sk', 'pk', 'id_hash', 'master_credential', 'master_signatures', 'event_credential', 'event_signatures']
            for prop in properties:
                if prop not in user_json:
                    return None
            _id = None
            if '_id' in user_json:
                _id = user_json['_id']
            return User(_id, user_json['email'], user_json['password'], user_json['sk'], user_json['pk'], user_json['id_hash'], user_json['master_credential'], user_json['master_signatures'], user_json['event_credential'], user_json['event_signatures'])

    def to_json(self):
        '''
        User object to json object
        NOTE: converts ObjectId to string
        '''
        obj = self.__dict__
        if obj['_id'] == None:
            del obj['_id']
        else:
            obj['_id'] = str(obj['_id'])
        return obj

    def get_all_users(self):
        '''
        Returns list of User objects from the database
        '''
        db = MongoWrapper().client['ticketing_store']
        coll = db['users']
        users = []
        for user_json in coll.find():
            user = User.from_json(user_json)
            users.append(user)
        return users

    @classmethod
    def insert_one(cls, user):
        '''
        Inserts a user object into the database
        '''
        json_obj = user.to_json()
        if json_obj != None:
            db = MongoWrapper().client['ticketing_store']
            coll = db['users']
            try:
                inserted = coll.insert_one(json_obj)
                return inserted.inserted_id
            except:
                return None

    @classmethod
    def find_user_by_attribute(cls, attribute, user_attribute):
        '''
        Finds a user by a specific attribute
        Returns user object
        '''
        db = MongoWrapper().client['ticketing_store']
        coll = db['users']
        user_json = coll.find_one({ attribute: user_attribute })

        if user_json:
            user = User.from_json(user_json)
            return user
        return None

    @classmethod
    def find_users_from_search(cls, query, page):
        """
        Returns users with matching name
        """
        PAGE_SIZE = 10
        filter = {
            '$or':
                [
                    {'master_credential': {'$regex': f".*{query}.*", '$options': 'i'}},
                    {'event_credentials': {'$regex': f".*{query}.*", '$options': 'i'}},
                    {'email': {'$regex': f".*{query}.*", '$options': 'i'}}
                ]
        }
        skip = (int(page)-1)*PAGE_SIZE
        limit = PAGE_SIZE

        db = MongoWrapper().client['ticketing_store']
        coll = db['users']
        results = coll.find(filter=filter,skip=skip,limit=limit).collation({'locale':'en'}).sort([('first_name',1),('last_name',1)])

        return [User.from_json(x) for x in results]

    @classmethod
    def update_user_attribute(cls, query_attribute, query_user_attribute, attribute, user_attribute):
        '''
        Queries for user by query_attribute = query_user_attribute
        Updates attribute of user to user_attribute
        '''
        query = { query_attribute: query_user_attribute }
        values = { "$set": { attribute: user_attribute } }
        db = MongoWrapper().client['ticketing_store']
        coll = db['users']
        coll.update_one(query, values)

    @classmethod
    def push_user_attribute(cls, query_attribute, query_user_attribute, attribute, user_attribute):
        '''
        Queries for user by query_attribute = query_user_attribute
        Appends attribute of user to user_attribute
        '''
        query = { query_attribute: query_user_attribute }
        values = { "$push": { attribute: user_attribute } }
        db = MongoWrapper().client['ticketing_store']
        coll = db['users']
        coll.update_one(query, values)

    @classmethod
    def update_user_attributes(cls, query_attribute, query_user_attribute, values):
        '''
        Queries for user by query_attribute = query_user_attribute
        Updates attribute of user to user_attribute
        '''
        query = { query_attribute: query_user_attribute }
        values = { "$set": values }
        db = MongoWrapper().client['ticketing_store']
        coll = db['users']
        coll.update_one(query, values)

    ########## Checking user validity ##########
    @classmethod
    def unused_email(cls, user_email):
        '''
        Determines if the supplied email is unused
        Returns True if email unused
        Returns False if email taken
        '''
        db = MongoWrapper().client['ticketing_store']
        coll = db['users']
        return(coll.find_one({ 'email': user_email }) is None)

    @classmethod
    def valid_email(cls, email):
        '''
        Checks that email is valid - per regex below
        Checks that email is not None. 
        Original code from:
        https://www.geeksforgeeks.org/check-if-email-address-valid-or-not-in-python/
        '''
        regex = r'^\w+([\.-]?\w+)*@\w+([\.-]?\w+)*(\.\w{2,3})+$'

        # Compares email to regex format
        if email is not None and re.search(regex, email):
            return True

        return False
    
    @classmethod
    def valid_password(cls, password):
        '''
        Checks that password is valid - > 7 characters
        Checks that password is not None
        '''
        if password is not None and len(password) > 7:
            return True

        return False

    @classmethod
    def valid_matching_passwords(cls, password, confirm_password):
        '''
        Checks that passwords match
        Checks that confirm password is not None
        '''
        return confirm_password is not None and password == confirm_password

    @classmethod
    def valid_name(cls, name):
        '''
        Checks that name is valid - between 1 and 50 characters
        Checks that name is not None. 
        Original code from:
        https://www.geeksforgeeks.org/check-if-email-address-valid-or-not-in-python/
        '''
        if name is not None and len(name) >= 1 and len(name) <= 50:
            return True

        return False

    @classmethod
    def is_valid_user(cls, user):
        '''
        Determines if user object is valid
        Returns True if valid, False otherwise
        '''
        attributes = ['_id', 'email', 'password', 'sk', 'pk', 'id_hash', 'master_credential', 'master_signatures', 'event_credential', 'event_signatures']
        for attribute in attributes:
            if not hasattr(user, attribute):
                return False
        return True

    @classmethod
    def is_valid_id(cls, _id):
        '''
        Determines if _id is 24-character hex string
        '''
        return len(_id) == 24