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
  Text,
  VStack,
  Wrap,
  WrapItem
} from "@chakra-ui/react";
import API from "../helpers/api";

const F3B = () => {

  const [encryptInput, setEncryptInput] = useState("");
  const [encryptOutput, setEncryptOutput] = useState("");
  const [decryptInput, setDecryptInput] = useState("");
  const [decryptOutput, setDecryptOutput] = useState("");
  const [idHash, setIdHash] = useState("");
  const [masterCredential, setMasterCredential] = useState("");
  const [eventName, setEventName] = useState("");
  const [idHash2, setIdHash2] = useState("");
  const [id, setId] = useState("");
  const [eventCredential, setEventCredential] = useState("");

  const [alertStatus, setAlertStatus] = useState("error");
  const [alertDisplay, setAlertDisplay] = useState("none");
  const [alertMessage, setAlertMessage] = useState("");

  // Handle encryption
  const handleEncryption = (e) => {
    e.preventDefault();
    const details = {
      message: encryptInput,
    };
    console.log("HANDLE ENCRYPTION");
    console.log(details);
    API.postAuthPath("dkg/encrypt", details)
      .then((json) => {
        console.log(json);
        setEncryptOutput(json.data.encrypted_message);
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

  // Handle encryption
  const handleDecryption = (e) => {
    e.preventDefault();
    const details = {
      encrypted_message: decryptInput,
    };
    console.log("HANDLE DECRYPTION");
    console.log(details);
    API.postAuthPath("dkg/decrypt", details)
      .then((json) => {
        console.log(json);
        setDecryptOutput(json.data.decrypted_message);
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

  // Handle issue master credential
  const handleIssueMasterCredential = (e) => {
    e.preventDefault();
    const details = {
      id: id,
    };
    console.log("HANDLE Issue Master Credential");
    console.log(details);
    API.postAuthPath("dkg/issue-master-credential", details)
      .then((json) => {
        console.log(json);
        setMasterCredential(json.data.master_credential);
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

  // Handle issue event credential
  const handleIssueEventCredential = (e) => {
    e.preventDefault();
    const details = {
      event_name: eventName,
      id_hash: idHash,
    };
    console.log("HANDLE Issue Event Credential");
    console.log(details);
    API.postAuthPath("dkg/issue-event-credential", details)
      .then((json) => {
        console.log(json);
        setEventCredential(json.data.event_credential);
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

  // Handle issue event credential
  const handleDKGAuthEvent = (e) => {
    e.preventDefault();
    const details = {
      event_name: eventName,
    };
    console.log("HANDLE DKG Auth Event");
    console.log(details);
    API.postAuthPath("dkg/auth-event-tx", details)
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

  return (
    <Flex pl="14%" pr="14%" direction="column" pb='40px'>
      <VStack>
        <Heading align="center" my="1rem" pb='20px'>Encrypt Input</Heading>
        <form onSubmit={handleEncryption}>
          <VStack>
            <Input
              bg="white"
              placeholder="Input to Encrypt"
              mb="10px"
              color="black"
              onChange={(e) => setEncryptInput(e.target.value)}
              width='300px'
            />
            <Button width='300px' type="submit" colorScheme="teal" mb="0px">
              Encrypt
            </Button>
          </VStack>
        </form>
        {
          encryptOutput.valueOf() !== "".valueOf() && 
          <Text>{encryptOutput}</Text>
        }
      </VStack>
      <VStack>
        <Heading align="center" my="1rem" pb='20px'>Decrypt Input</Heading>
        <form onSubmit={handleDecryption}>
          <VStack>
            <Input
              bg="white"
              placeholder="Input to Decrypt"
              mb="10px"
              color="black"
              onChange={(e) => setDecryptInput(e.target.value)}
              width='300px'
            />
            <Button width='300px' type="submit" colorScheme="teal" mb="0px">
              Decrypt
            </Button>
          </VStack>
        </form>
        {
          decryptOutput.valueOf() !== "".valueOf() && 
          <Text>{decryptOutput}</Text>
        }
      </VStack>
      <VStack>
        <Heading align="center" my="1rem" pb='20px'>Issue Master Credential</Heading>
        <form onSubmit={handleIssueMasterCredential}>
          <VStack>
            <Input
              bg="white"
              placeholder="Input Name"
              mb="10px"
              color="black"
              onChange={(e) => setId(e.target.value)}
              width='300px'
            />
            <Button width='300px' type="submit" colorScheme="teal" mb="0px">
              Issue Master Credential
            </Button>
          </VStack>
        </form>
        {
          masterCredential.valueOf() !== "".valueOf() && 
          <Text>{masterCredential}</Text>
        }
      </VStack>
      <VStack>
        <Heading align="center" my="1rem" pb='20px'>Issue Event Credential</Heading>
        <form onSubmit={handleIssueEventCredential}>
          <VStack>
            <Input
              bg="white"
              placeholder="Input Name"
              mb="10px"
              color="black"
              onChange={(e) => setIdHash(e.target.value)}
              width='300px'
            />
            <Input
              bg="white"
              placeholder="Input Event Name"
              mb="10px"
              color="black"
              onChange={(e) => setEventName(e.target.value)}
              width='300px'
            />
            <Button width='300px' type="submit" colorScheme="teal" mb="0px">
              Issue Event Credential
            </Button>
          </VStack>
        </form>
        {
          eventCredential.valueOf() !== "".valueOf() && 
          <Text>{eventCredential}</Text>
        }
      </VStack>
      <VStack>
        <Heading align="center" my="1rem" pb='20px'>Is user authorized for event?</Heading>
        <form onSubmit={handleDKGAuthEvent}>
          <VStack>
            <Input
              bg="white"
              placeholder="Input Event Name"
              mb="10px"
              color="black"
              onChange={(e) => setEventName(e.target.value)}
              width='300px'
            />
            <Button width='300px' type="submit" colorScheme="teal" mb="0px">
              Is user authorized for event?
            </Button>
          </VStack>
        </form>
        {
          eventCredential.valueOf() !== "".valueOf() && 
          <Text>{eventCredential}</Text>
        }
      </VStack>
    </Flex>
    );
};

export default F3B;