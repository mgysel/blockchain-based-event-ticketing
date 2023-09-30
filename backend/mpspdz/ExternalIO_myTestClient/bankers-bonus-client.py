#!/usr/bin/python3

import sys

sys.path.append('.')

from client import *
from domains import *

client_id = int(sys.argv[1])
n_parties = int(sys.argv[2])
bonus = float(sys.argv[3])
finish = int(sys.argv[4])

client = Client(['localhost'] * n_parties, 14000, client_id)

type = client.specification.get_int(4)

if type == ord('R'):
    domain = Z2(client.specification.get_int(4))
elif type == ord('p'):
    domain = Fp(client.specification.get_bigint())
else:
    raise Exception('invalid type')

for socket in client.sockets:
    os = octetStream()
    os.store(finish)
    os.Send(socket)

for x in bonus, bonus * 2 ** 16:
    client.send_private_inputs([domain(x)])

    print('Winning client id is :',
          client.receive_outputs(domain, 1)[0].v % 2 ** 64)
    


# PLAYERS=<nparties> Scripts/<protocol>.sh bankers_bonus-1 & ./bankers-bonus-client.x 0 <nparties> 100 0 & ./bankers-bonus-client.x 1 <nparties> 200 0 & ./bankers-bonus-client.x 2 <nparties> 50 1

# PLAYERS=2 Scripts/mascot.sh bankers_bonus-1 & ./bankers-bonus-client.x 0 2 100 0 & ./bankers-bonus-client.x 1 2 200 0 & ./bankers-bonus-client.x 2 2 50 1


# PLAYERS=2 Scripts/mascot.sh bankers_bonus-1 & python ExternalIO/bankers-bonus-client.py 0 2 100 0 & python ExternalIO/bankers-bonus-client.py 1 2 200 0 & python ExternalIO/bankers-bonus-client.py 2 2 50 1
