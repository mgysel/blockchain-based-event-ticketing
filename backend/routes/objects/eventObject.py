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

class Event:
    '''
    Event class that contains basic event info/methods
    '''
    def __init__(self, _id, owner, name, num_tickets, price, max_resale_price, resale_royalty, n_rebuy_tx):
        self._id = _id
        self.owner = owner
        self.name = name
        self.num_tickets = num_tickets
        self.price = price
        self.max_resale_price = max_resale_price
        self.resale_royalty = resale_royalty
        self.n_rebuy_tx = n_rebuy_tx

    def from_json(event_json):
        '''
        Event json object to Event Object
        '''
        if event_json != None:
            properties = ['owner', 'name', 'num_tickets', 'price', 'max_resale_price', 'resale_royalty', 'n_rebuy_tx']
            for prop in properties:
                if prop not in event_json:
                    return None
            _id = None
            if '_id' in event_json:
                _id = event_json['_id']
            return Event(_id, event_json['owner'], event_json['name'], event_json['num_tickets'], event_json['price'], event_json['max_resale_price'], event_json['resale_royalty'], event_json['n_rebuy_tx'])

    def to_json(self):
        '''
        Event object to json object
        NOTE: converts ObjectId to string
        '''
        obj = self.__dict__
        if obj['_id'] == None:
            del obj['_id']
        else:
            obj['_id'] = str(obj['_id'])
        return obj
    
    def to_json_str(self):
        '''
        Converts one User to a string.
        '''
        obj = self.__dict__
        if obj['_id'] == None:
            del obj['_id']
        else:
            obj['_id'] = str(obj['_id'])
        return obj
    
    @staticmethod
    def many_to_json(events):
        '''
        Converts a list of events to a list of python inbuilt objects.
        '''
        l = []
        for event in events:
            l.append(event.to_json())
        return l

    @staticmethod
    def many_to_json_str(events):
        '''
        Converts a list of Events to a list of strings.
        '''
        l = []
        for event in events:
            l.append(event.to_json_str())
        return l   

    @classmethod
    def get_all_events(cls):
        '''
        Returns list of Event objects from the database
        '''
        db = MongoWrapper().client['ticketing_store']
        coll = db['events']
        events = []
        for event_json in coll.find():
            event = Event.from_json(event_json)
            events.append(event)
        return events

    @classmethod
    def insert_one(cls, event):
        '''
        Inserts an event object into the database
        '''
        json_obj = event.to_json()
        if json_obj != None:
            db = MongoWrapper().client['ticketing_store']
            coll = db['events']
            try:
                inserted = coll.insert_one(json_obj)
                return inserted.inserted_id
            except:
                return None

    @classmethod
    def find_event_by_attribute(cls, attribute, event_attribute):
        '''
        Finds an event by a specific attribute
        Returns Event object
        '''
        db = MongoWrapper().client['ticketing_store']
        coll = db['events']
        event_json = coll.find_one({ attribute: event_attribute })

        if event_json:
            event = Event.from_json(event_json)
            return event
        return None

    @classmethod
    def update_event_attribute(cls, query_attribute, query_event_attribute, attribute, event_attribute):
        '''
        Queries for event by query_attribute = query_event_attribute
        Updates attribute of event to event_attribute
        '''
        query = { query_attribute: query_event_attribute }
        values = { "$set": { attribute: event_attribute } }
        db = MongoWrapper().client['ticketing_store']
        coll = db['events']
        coll.update_one(query, values)

    @classmethod
    def push_event_attribute(cls, query_attribute, query_event_attribute, attribute, event_attribute):
        '''
        Queries for event by query_attribute = query_event_attribute
        Appends attribute of event to event_attribute
        '''
        query = { query_attribute: query_event_attribute }
        values = { "$push": { attribute: event_attribute } }
        db = MongoWrapper().client['ticketing_store']
        coll = db['events']
        coll.update_one(query, values)

    @classmethod
    def update_event_attributes(cls, query_attribute, query_event_attribute, values):
        '''
        Queries for event by query_attribute = query_event_attribute
        Updates attribute of event to event_attribute
        '''
        query = { query_attribute: query_event_attribute }
        values = { "$set": values }
        db = MongoWrapper().client['ticketing_store']
        coll = db['events']
        coll.update_one(query, values)