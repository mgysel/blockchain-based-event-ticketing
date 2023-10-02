
# Blockchain-based Event Ticketing

  

## Project Description

  

Event Ticketing is a $78 Billion industry globally, with the secondary ticket market accounting for $19 Billion of this. However, event organizers have no control over the secondary ticket market and attendees experience high rates of ticket fraud. In fact, Ticketmaster estimates that bots siphon off 60% of tickets for major events that are then resold at higher prices. Event organizers have no control over these secondary ticket prices and see none of the resale value. Furthermore, 12% of adults in the United States have purchased fraudulent tickets online, creating a lack of trust in the secondary ticket market. Thus, the inability of event organizers to control the secondary ticket market for their own events and the inability of attendees to trust the secondary ticket market present major challenges in existing event ticketing systems.

This project implements a blockchain-based event ticketing system with the goals of giving event organizers control over the secondary ticket market and preventing ticket fraud. This is achieved through the use of an identification system and ticket market. A more in-depth discussion of the project can be viewed in `Final_Report.pdf` and `Final_Presentation.pdf`, and a demo video can be seen below.

### Demo Video

https://github.com/mgysel/blockchain-based-event-ticketing/assets/18092117/bc1ed762-977f-4751-a566-abb02d32b513

### Identification System

The identification system allows user identities to be linked to event tickets in a privacy-preserving manner. Prior to purchasing an event ticket, ticket buyers scan their identification card and gain anonymous event credentials that can be used to purchase event tickets. When purchasing an event ticket, ticket buyers submit their payment and event credential, which is stored in the event ticket. At the event, attendees scan their event ticket and identification card, where the event credential is again computed from the identification card. The event organizer admits entrance if the event ticket is valid and contains the matching event credential.

  

### Ticket Market

The ticket market utilizes an event smart contract that allows event organizers to create events and sell tickets and attendees to buy, resell, and use tickets. In the primary ticket market, ticket buyers purchase event tickets directly through the event smart contract. In the secondary ticket market, ticket resellers and buyers utilize Flash Freezing Flash Boys (F3B), which relies on a commit-reveal scheme that enables encrypted transactions and delayed execution. Thus, secondary ticket market resellers and buyers commit to encrypted transactions, which are later shuffled, decrypted, and executed, thereby sending the ticket resellers the ticket payment and the ticket buyers the event tickets.

## Global Architecture

The blockchain-based event ticketing system was implemented using a ReactJS web frontend, Flask web backend, MongoDB database, Dela blockchain, CanDID identification system credential-issuance committee, and F3B secret-management committee. 

### Web Frontend
The web frontend was built with ReactJS, allowing users to interact with these other systems. This allows an event organizer to create and event and sell tickets, and attendees to buy and resell event tickets.

### Web Backend
The web backend was built with Flask. This receives requests from the web frontend, interacts with the database, blockchain, credential-issuance committee, and secret-management committee, and relays responses back to the web frontend.

### MongoDB Database
MongoDB was used as a traditional database in order to store basic user profile and event information.

### Dela blockchain
The event smart contract was built on the Dela blockchain. Thus, event organizers submit transactions to Dela to initiate an event smart contract. Ticket buyers and resellers submit transactions to Dela to buy and sell event tickets on the primary and secondary ticket markets.

### CanDID Identification System credential-issuance committee
The credential-issuance committee is a decentralized collection of nodes that users interact with to obtain event credentials that are used to purchase event tickets. Users send the credential-issuance committee secret shares of their identification card information, which then computes a hash of the user's identification in secure-multiparty computation. This allows the credential-issuance committee to deduplicate identities without learning any information of the identity. Secondly, users interact with the credential-issuance committee to obtain event credentials, which attach the event tickets to the user's identity in a privacy-preserving manner.  

### F3B secret-management committee
The Flash Freezing Flash Boys (F3B) secret-management committee allows for transactions to be encrypted and then later decrypted and executed. This secret-management committee was used to allow secondary ticket market transactions to be encrypted and later executed. Thus, when tickets for an event are sold out, ticket resellers and buyers encrypt resell and buy transactions; moreover, at a set time before the event, these transactions are shuffled, decrypted, and executed.

## Project Setup

Prior to running the blockchain-based event ticketing system, the web frontend, web backend, and Dela blockchain must be setup and dependencies installed.

  

### Setup Web Frontend

The frontend React application can be found in the `frontend` directory. Install all dependencies as follows:

```sh

$  cd frontend
$  npm  install

```

  

### Setup Web Backend

The backend Flask application can be found in the `backend` directory. Install all dependencies as follows:

```sh
$  cd backend
$  pip3  install  -r  requirements.txt

```

  

### Setup Dela

1: Install [Go](https://go.dev/dl/) v1.18.

  

2: Install the `crypto` utility from Dela:

  

```sh

cd  backend/dela/cli/crypto

go  install

```

  

Go will install the binaries in `$GOPATH/bin`, so be sure this it is correctly

added to you path (e.g. `export PATH=$PATH:/Users/username/go/bin`).

### Setup Number of Nodes and Ports

The ports used for the backend server, dela nodes, and identification system nodes can be found and updated in `backend/setup_info.json`. The ports used for the frontend can be found in `frontend/src/helpers/api.js`. Ensure the server_port in `backend/setup_info.json` matches this.url found in `frontend/src/helpers/api.js` so the frontend and backend can communicate.

## Run the Project

  

To run the project, open four separate terminals, which will be used for the Dela blockchain, identification system, frontend, and backend, respectively.

In the first terminal, run the dela blockchain as follows
```sh
$  cd backend
$  python3 setup_dela.py
```

In the second terminal, run the identification system credential-issuance committee as follows
  ```sh
$  cd backend
$  python3 setup_dkg.py
```

In the third terminal, run the web backend server as follows
  ```sh
$  cd backend
$  python3 server.py
```

In the fourth terminal, run the web frontend as follows
  ```sh
$  cd frontend
$  npm start
```

This will run and automatically launch the web application on `http://localhost:3000`. Now you can create events and buy and sell tickets on the primary and secondary ticket markets!

## Project Files

  

The project folder structure and files are outlined below:

* backend
	* credentials
		* credentials.json: This file contains the connection string to connect to the MongoDB database instance, and should thus be kept secret.
	* dela
		* contracts
			* event: The event folder contains the event smart contract used for the primary and secondary ticket markets, the event smart contract controller, and unit testing for both the smart contract and controller.
		* dkg: The dkg folder contains the F3B secret-management committee and identification system credential issuance committee implementations. The existing dkg infrastructure was used to apply additional credential issuance committee functionality including issuing and verifying the master and event credentials.
		* test: The test folder contains integration tests for the event smart contract as well as throughput measurement tests for the primary and secondary ticket markets.
	* mpspdz: The mpspdz folder contains the implementation of the user identification deduplication, specifically computing a hash of the user's identification using secure multi-party computation.
		* Player-Data: The Player-Data folder is where user identification secret shares are written to so that MP-SPDZ nodes can read from these files.
		* Programs/Source
			* prf_mimc_mine.mpc: This file contains the implementation of the MiMC Pseudo-Random Function, used to compute the hash of the user's identification.
	* routes
		* routes: The routes folder handles all requests sent to the web backend server and returns responses. These routes receive HTTP Requestsfrom the web frontend; communicate with the identification system, secret-mangement committee, and Dela blockchain; and return HTTP Responses to the web frontend.
		* objects: The objects folder contains event and user classes that are used to interact with the MongoDB database event and user tables, respectively.
	* setup_dela.py: setup_dela.py is run to start the Dela blockchain.
	* setup_dkg.py: setup_dkg.py is run to start the secret-management committee and credential issuance committees.
	* server.py: server.py is run to start the web backend server.

* frontend
	* src
		* pages: The pages folder contains the frontend web pages, such as the landing page, homepage, and admin page.
		* components: The components folder contains React components reused throughout the web frontend.
		* helpers: The helpers folder contains `api.js` which is used to send HTTP Requests to the web backend and read HTTP Responses. It also contains `context.js`, which is the global storage for the frontend.
