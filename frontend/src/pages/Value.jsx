import React, { useEffect, useState } from "react";
import { useLocation, Link as RouterLink } from "react-router-dom";

import {
  Button,
  Heading,
  Flex,
  Grid,
  GridItem,
  HStack,
  Input,
  LinkBox,
  LinkOverlay,
  Table,
  TableContainer,
  TableCaption,
  Thead,
  Tbody,
  Td,
  Th,
  Tr,
  Text,
  VStack,
  Wrap,
  WrapItem
} from "@chakra-ui/react";
import API from "../helpers/api";

const ValueContract = () => {

  const [writeInputKey, setWriteInputKey] = useState("");
  const [writeInputValue, setWriteInputValue] = useState("");
  const [readInputKey, setReadInputKey] = useState("");
  const [readOutput, setReadOutput] = useState("");
  const [deleteInputKey, setDeleteInputKey] = useState("");
  const [listOutput, setListOutput] = useState([]);

  const [alertStatus, setAlertStatus] = useState("error");
  const [alertDisplay, setAlertDisplay] = useState("none");
  const [alertMessage, setAlertMessage] = useState("");

  // Handle write 
  const handleWrite = (e) => {
    e.preventDefault();
    const details = {
      key: writeInputKey,
      value: writeInputValue,
    };
    console.log("HANDLE WRITE");
    console.log(details);
    API.postAuthPath("sc/value/write", details)
      .then((json) => {
        console.log(json);
      })
      .catch((err) => {
        console.log("Error: ", err)
        err.json().then((json) => {
          setAlertDisplay("flex");
          setAlertStatus("error");
          setAlertMessage(json.message);
        });
      });
  };

  // Handle delete
  const handleDelete = (e) => {
    e.preventDefault();
    const details = {
      key: deleteInputKey,
    };
    console.log("HANDLE DELETE");
    console.log(details);
    API.postAuthPath("sc/value/delete", details)
      .then((json) => {
        console.log(json);
      })
      .catch((err) => {
        console.log("Error: ", err)
        err.json().then((json) => {
          setAlertDisplay("flex");
          setAlertStatus("error");
          setAlertMessage(json.message);
        });
      });
  };

  // Handle read
  const handleRead = (e) => {
    e.preventDefault();
    console.log("HANDLE READ");
    console.log("Read Input Key: ", readInputKey)
    API.getPath(`sc/value/read?key=${readInputKey}`)
      .then((json) => {
        console.log(json);
        setReadOutput(json.data.value);
      })
      .catch((err) => {
        console.log("Error: ", err)
        err.json().then((json) => {
          setAlertDisplay("flex");
          setAlertStatus("error");
          setAlertMessage(json.message);
        });
      });
  };

  // Handle list
  const handleList = (e) => {
    e.preventDefault();
    console.log("HANDLE LIST");
    API.getPath('sc/value/list')
      .then((json) => {
        console.log("List Output");
        console.log(json);
        setListOutput([]);
        setListOutput(json.data.pairs);
      })
      .catch((err) => {
        console.log("Error: ", err)
        err.json().then((json) => {
          setAlertDisplay("flex");
          setAlertStatus("error");
          setAlertMessage(json.message);
        });
      });
  };

  return (
    <Flex pl="14%" pr="14%" direction="column" pb='40px' pt='40px'>
      <VStack border='1px solid black' borderRadius='5px' p='20px' mb='20px'>
        <Heading align="center" my="1rem">Write to Dela</Heading>
        <form onSubmit={handleWrite}>
          <VStack>
            <Input
              bg="white"
              placeholder="Key"
              mb="10px"
              color="black"
              onChange={(e) => setWriteInputKey(e.target.value)}
              width='300px'
            />
            <Input
              bg="white"
              placeholder="Value"
              mb="10px"
              color="black"
              onChange={(e) => setWriteInputValue(e.target.value)}
              width='300px'
            />
            <Button width='300px' type="submit" colorScheme="teal" mb="0px">
              Write to Dela
            </Button>
          </VStack>
        </form>
        {
          writeInputKey.valueOf() !== "".valueOf() && 
          <>
            <Text>Key: {writeInputKey}</Text>
            <Text>Value: {writeInputValue}</Text>
          </>
        }
      </VStack>
      <VStack border='1px solid black' borderRadius='5px' p='20px' mb='20px'>
        <Heading align="center" my="1rem" >Read from Dela</Heading>
        <form onSubmit={handleRead}>
          <VStack>
            <Input
              bg="white"
              placeholder="Key to read from Dela"
              mb="10px"
              color="black"
              onChange={(e) => setReadInputKey(e.target.value)}
              width='300px'
            />
            <Button width='300px' type="submit" colorScheme="teal" mb="0px">
              Read from Dela
            </Button>
          </VStack>
        </form>
        {
          readOutput.valueOf() !== "".valueOf() && 
          <Text>{readOutput}</Text>
        }
      </VStack>
      <VStack border='1px solid black' borderRadius='5px' p='20px' mb='20px'>
        <Heading align="center" my="1rem">Delete from Dela</Heading>
        <form onSubmit={handleDelete}>
          <VStack>
            <Input
              bg="white"
              placeholder="Key"
              mb="10px"
              color="black"
              onChange={(e) => setDeleteInputKey(e.target.value)}
              width='300px'
            />
            <Button width='300px' type="submit" colorScheme="teal" mb="0px">
              Delete from Dela
            </Button>
          </VStack>
        </form>
      </VStack>
      <VStack border='1px solid black' borderRadius='5px' p='20px' mb='20px'>
        <Heading align="center" my="1rem">List from Dela</Heading>
        <form onSubmit={handleList}>
          <VStack>
            <Button width='300px' type="submit" colorScheme="teal" mb="0px">
              List Key-Value Pairs from Dela
            </Button>
          </VStack>
        </form>
        {
          listOutput.length !== 0 && 
          <VStack width='80%'>
            <Heading align="center" my="1rem" pb='20px'>
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
                {listOutput.map((pair, i) => (
                  <Tr index={i}>
                    <Td>{pair.key}</Td>
                    <Td>{pair.value}</Td>
                  </Tr>
                ))}
                </Tbody>
              </Table>
            </TableContainer>
          </VStack>
        }
      </VStack>
    </Flex>
    );
};

export default ValueContract;