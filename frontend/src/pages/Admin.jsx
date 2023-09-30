import React, { useEffect, useState, useContext } from "react";
import { useLocation } from "react-router-dom";
import {
  Box,
  Button,
  Center,
  Grid,
  GridItem,
  Heading,
  Flex,
  Text,
  VStack,
} from "@chakra-ui/react";
import {
  Table,
  Thead,
  Tbody,
  Tfoot,
  Tr,
  Th,
  Td,
  TableCaption,
  TableContainer,
} from '@chakra-ui/react'
import API from "../helpers/api";
import { StoreContext } from "../helpers/context";

const Admin = () => {

  const context = useContext(StoreContext);
  const [eventName, setEventName] = useState("");
  console.log("Event Name: ", eventName)
  const [ownedTickets, setOwnedTickets] = useState([]);
  const [resaleTickets, setResaleTickets] = useState([]);
  const [rebuyTickets, setRebuyTickets] = useState([]);
  const [userBalances, setUserBalances] = useState([]);
  const [numRebuyTxs, setNumRebuyTxs] = useState(0);
  const [keyValueList, setKeyValueList] = useState([]);
  console.log("Owned tickets: ", ownedTickets);
  console.log("Resale Tickets: ", resaleTickets);
  console.log("Rebuy Tickets: ", rebuyTickets);
  console.log("User Balances: ", userBalances);

  const [alertStatus, setAlertStatus] = useState("error");
  const [alertDisplay, setAlertDisplay] = useState("none");
  const [alertMessage, setAlertMessage] = useState("");

  // Get all owned tickets
  useEffect(() => {
    API.getPath("sc/event/read")
      .then((json) => {
        console.log("ADMIN - Get Event Contract Data");
        console.log(json);
        context.contractData[1](json.data.contract_data);
        context.contractData[0] = json.data.contract_data;
        const contractData = context.contractData[0];
        console.log("Contract data: ", context.contractData[0]);
        setOwnedTickets([])
        setOwnedTickets([...json.data.contract_data.owners]);
        setResaleTickets([])
        setResaleTickets([...json.data.contract_data.resellers]);
        setRebuyTickets([])
        setRebuyTickets([...contractData.rebuyers]);
        setUserBalances([])
        setUserBalances([...contractData.users_balance]);
        setEventName(contractData.event_name);
      })
      .catch((err) => {
        console.warn(`Error: ${err}`);
      });
  }, []);

  useEffect(() => {
    console.log("ADMIN - HANDLE READ");
    const readInputKey = 'n_rebuy_txs';
    API.getPath(`sc/value/read?key=${readInputKey}`)
      .then((json) => {
        console.log("")
        console.log(json);
        setNumRebuyTxs(json.data.value);
      })
      .catch((err) => {
        console.log("Error: ", err)
        err.json().then((json) => {
          setAlertDisplay("flex");
          setAlertStatus("error");
          setAlertMessage(json.message);
        });
      });
  }, []);

  useEffect(() => {
    console.log("ADMIN - Get Value Contract List");
    API.getPath("sc/value/list")
      .then((json) => {
        console.log("Value ListSuccess")
        console.log(json);
        console.log(json.data.pairs);
        setKeyValueList([])
        setKeyValueList([...json.data.pairs]);
      })
      .catch((err) => {
        console.log("Error: ", err)
        err.json().then((json) => {
          setAlertDisplay("flex");
          setAlertStatus("error");
          setAlertMessage(json.message);
        });
      });
  }, []);
  
  // Handle Shuffle, Decrypt, Execute Secondary Batch
  const handleSecondaryBatch = (e) => {
    e.preventDefault();
    console.log("Inside handleSecondaryBatch")
    console.log("Event Name: ", eventName)

    API.getPath(`sc/event/decrypt-execute-secondary?name=${eventName}`)
      .then((json) => {
        console.log("Success");
        console.log(json)
      })
      .catch((err) => {
        err.json().then((json) => {
          console.log("Error")
          console.log(json)
        });
      });
  };

  // Execute Secondary Transaction
  const handleExecuteSecondaryBatch = (e) => {
    e.preventDefault();
    console.log("Inside handleExecuteSecondaryBatch")
    console.log("Event Name: ", eventName)

    API.getPath(`sc/event/transact-secondary?name=${eventName}`)
      .then((json) => {
        console.log("Success");
        console.log(json)
      })
      .catch((err) => {
        err.json().then((json) => {
          console.log("Error")
          console.log(json)
        });
      });
  };

  return (
    <Flex pl="14%" pr="14%" direction="column" pb='40px'>
      <Heading align="center" my="1rem">
        Admin Page
      </Heading>
      <Flex pl="14%" pr="14%" direction="column" pb='40px' border='1px solid black' borderRadius='5px' mb='20px'>
        <Heading align="center" my="1rem" pb='20px'>Handle Secondary Transactions</Heading>
        <VStack>
          <Button colorScheme="teal" mb="0px" onClick={handleSecondaryBatch}>
            Shuffle, Decrypt, Execute Resell and Rebuy TX
          </Button>
          <Button colorScheme="teal" mb="0px" onClick={handleExecuteSecondaryBatch}>
            Transfer Secondary Market Tickets
          </Button>
        </VStack>
      </Flex>
      <Flex pl="14%" pr="14%" direction="column" pb='40px' border='1px solid black' borderRadius='5px' mb='20px'>
        <VStack border='1px solid black' borderRadius='5px' p='10px' m='10px'>
          <Heading align="center" my="1rem">
              Value Contract Storage
          </Heading>
          <TableContainer>
            <Table variant='simple' size='lg'>
              <TableCaption>Table of key-value pairs stored in value contract</TableCaption>
              <Thead>
                <Tr>
                  <Th>Key</Th>
                  <Th>Value</Th>
                </Tr>
              </Thead>
              <Tbody>
              {keyValueList.map((pair, i) => (
                <Tr index={i}>
                  <Td>{pair.key}</Td>
                  <Td>{pair.value}</Td>
                </Tr>
              ))}
              </Tbody>
            </Table>
          </TableContainer>
        </VStack>
        <VStack border='1px solid black' borderRadius='5px' p='10px' m='10px'>
          <Heading align="center" my="1rem">
              Tickets owned
          </Heading>
          <TableContainer>
            <Table variant='simple' size='lg'>
              <TableCaption>Table of number of tickets owned for each public key</TableCaption>
              <Thead>
                <Tr>
                  <Th>Number of Tickets</Th>
                  <Th>Public Key</Th>
                </Tr>
              </Thead>
              <Tbody>
              {ownedTickets.map((ticket, i) => (
                <Tr index={i}>
                  <Td>{ticket.num_tickets}</Td>
                  <Td>{ticket.pk}</Td>
                </Tr>
              ))}
              </Tbody>
            </Table>
          </TableContainer>
        </VStack>
        <VStack border='1px solid black' borderRadius='5px' p='10px' m='10px'>
          <Heading align="center" my="1rem">
              Resale Tickets
          </Heading>
          <TableContainer>
            <Table variant='simple' size='lg'>
              <TableCaption>Table of number of resell proposals for each public key</TableCaption>
              <Thead>
                <Tr>
                  <Th>Number of Tickets</Th>
                  <Th>Ticket Price</Th>
                  <Th>Public Key</Th>
                </Tr>
              </Thead>
              <Tbody>
              {resaleTickets.map((ticket, i) => (
                <Tr index={i}>
                  <Td>{ticket.num_tickets}</Td>
                  <Td>{ticket.price}</Td>
                  <Td>{ticket.pk}</Td>
                </Tr>
              ))}
              </Tbody>
            </Table>
          </TableContainer>
        </VStack>
        <VStack border='1px solid black' borderRadius='5px' p='10px' m='10px'>
          <Heading align="center" my="1rem">
              Rebuy Tickets
          </Heading>
          <TableContainer>
            <Table variant='simple' size='lg'>
              <TableCaption>Table of number of rebuy proposals for each public key</TableCaption>
              <Thead>
                <Tr>
                  <Th>Number of Tickets</Th>
                  <Th>Ticket Price</Th>
                  <Th>Public Key</Th>
                </Tr>
              </Thead>
              <Tbody>
              {rebuyTickets.map((ticket, i) => (
                <Tr index={i}>
                  <Td>{ticket.num_tickets}</Td>
                  <Td>{ticket.price}</Td>
                  <Td>{ticket.pk}</Td>
                </Tr>
              ))}
              </Tbody>
            </Table>
          </TableContainer>
        </VStack>
        <VStack border='1px solid black' borderRadius='5px' p='10px' m='10px'>
          <Heading align="center" my="1rem">
              User Balances
          </Heading>
          <TableContainer>
            <Table variant='simple' size='lg'>
              <TableCaption>Table of number of tickets owned for each public key</TableCaption>
              <Thead>
                <Tr>
                  <Th>Balance</Th>
                  <Th>Public Key</Th>
                </Tr>
              </Thead>
              <Tbody>
              {userBalances.map((ticket, i) => (
                <Tr index={i}>
                  <Td>{ticket.balance}</Td>
                  <Td>{ticket.pk}</Td>
                </Tr>
              ))}
              </Tbody>
            </Table>
          </TableContainer>
        </VStack>
      </Flex>
    </Flex>
  );
};

export default Admin;