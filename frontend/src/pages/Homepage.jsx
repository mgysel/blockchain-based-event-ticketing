import React, { useEffect, useState, useContext } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import {
  Box,
  Button,
  Center,
  Grid,
  GridItem,
  Heading,
  HStack,
  Input,
  Flex,
  Text,
} from "@chakra-ui/react";
import API from "../helpers/api";
import { StoreContext } from "../helpers/context";

const Homepage = () => {

  const navigate = useNavigate();
  const context = useContext(StoreContext);
  const setContractData = context.contractData[1];

  const [userMasterCredential, setUserMasterCredential] = useState("");
  const [userPk, setUserPk] = useState("");
  const [userIdHash, setUserIdHash] = useState("");
  const [idInput, setIdInput] = useState("");

  const [eventName, setEventName] = useState("");
  const [numTickets, setNumTickets] = useState("0");
  const [numTicketsForResale, setNumTicketsForResale] = useState("0");
  const [numTicketsForRebuy, setNumTicketsForRebuy] = useState("0");
  const [resellTickets, setResellTickets] = useState(false);
  const [useTickets, setUseTickets] = useState(false);

  const [resellNumTickets, setResellNumTickets] = useState("");
  const [resellPrice, setResellPrice] = useState("");

  const [useTicketNumTickets, setUseTicketNumTickets] = useState("");
  const [useTicketID, setUseTicketID] = useState("");

  const [alertStatus, setAlertStatus] = useState("error");
  const [alertDisplay, setAlertDisplay] = useState("none");
  const [alertMessage, setAlertMessage] = useState("");

  // Get user profile
  useEffect(() => {
    API.getPath("user/profile")
      .then((json) => {
        console.log(json)
        setUserMasterCredential(json.data.master_credential);
        context.masterCredential[0] = json.data.master_credential;
        setUserPk(json.data.pk);
        context.PK[0] = json.data.pk;
        setUserIdHash(json.data.id_hash);
        context.idHash[0] = json.data.id_hash;
        
      })
      .catch((err) => {
        console.warn(`Error: ${err}`);
      });
  }, []);

  // Get contract data
  useEffect(() => {
    API.getPath("sc/event/read")
      .then((json) => {
        console.log("Getting event contract data");
        console.log(json);
        context.contractData[1](json.data.contract_data);
        context.contractData[0] = json.data.contract_data;
        const contractData = context.contractData[0];
        console.log("Contract data: ", context.contractData[0]);

        const userPk = context.PK[0];

        if (contractData != null && userPk.valueOf() !== "".valueOf()) {
          // Get event name
          setEventName(json.data.contract_data.event_name);
          // Get number of tickets owned by user
          for (let i=0; i < json.data.contract_data.owners.length; i++) {
            if (json.data.contract_data.owners[i].pk === userPk) {
              setNumTickets(json.data.contract_data.owners[i].num_tickets);
              break;
            }
          }
          // Get number of tickets for resale
          for (let i=0; i < json.data.contract_data.resellers.length; i++) {
            if (json.data.contract_data.resellers[i].pk === userPk) {
              setNumTicketsForResale(json.data.contract_data.resellers[i].num_tickets);
              break;
            }
          }
          // Get number of tickets for rebuy
          for (let i=0; i < json.data.contract_data.rebuyers.length; i++) {
            if (json.data.contract_data.rebuyers[i].pk === userPk) {
              setNumTicketsForRebuy(json.data.contract_data.rebuyers[i].num_tickets);
              break;
            }
          }
        }
      })
      .catch((err) => {
        console.warn(`Error: ${err}`);
      });
  }, [userPk]);

  const handleClickResellOpen = () => {
    setResellTickets(true);
  }

  const handleClickResellClose = () => {
    setResellTickets(false);
  }

  const handleClickUseTicketsOpen = () => {
    setUseTickets(true);
  }

  const handleClickUseTicketsClose = () => {
    setUseTickets(false);
  }

  // Handle reselling tickets
  const handleResellTickets = (e) => {
    const details = {
      event_name: eventName,
      num_tickets: resellNumTickets,
      price: resellPrice,
    };
    console.log("HANDLE RESELL Tickets")
    console.log(details)
    API.postAuthPath("sc/event/resell", details)
      .then((json) => {
        console.log(json)
        navigate("/home");
      })
      .catch((err) => {
        console.log("Error: ", err)
        err.json().then((json) => {
          setAlertDisplay("flex");
          setAlertStatus("error");
          setAlertMessage(json.message);
        });
        navigate("/home");
      });
  };

  // Handle use tickets
  const handleUseTickets = (e) => {
    const details = {
      event_name: eventName,
      num_tickets: useTicketNumTickets,
      id: useTicketID,
    };
    console.log("HANDLE USE TICKETS")
    console.log("Use num tickets: ", useTicketNumTickets)
    console.log(details)
    API.postAuthPath("sc/event/use-ticket", details)
      .then((json) => {
        console.log(json)
        navigate("/home");
      })
      .catch((err) => {
        console.log("Error: ", err)
        err.json().then((json) => {
          setAlertDisplay("flex");
          setAlertStatus("error");
          setAlertMessage(json.message);
        });
        navigate("/home");
      });
  };

  // Handle issue master credential
  const handleIssueMasterCredential = (e) => {
    e.preventDefault();
    const details = {
      id: idInput,
    };
    console.log("HANDLE Issue Master Credential");
    console.log(details);
    API.postAuthPath("dkg/issue-master-credential", details)
      .then((json) => {
        console.log(json);
        setUserMasterCredential(json.data.master_credential);
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
      <Heading align="center" my="1rem">
        Homepage
      </Heading>
      {userMasterCredential.length === 0?
        <Flex pl="14%" pr="14%" direction="column" pb='40px' border='1px solid black' borderRadius='5px' mb='20px'>
          <Heading align="center" my="1rem" pb='20px'>Get Master Credential</Heading>
          <Text pb='10px'>Please enter your identification before purchasing tickets</Text>
          <form onSubmit={handleIssueMasterCredential}>
            <Input
              bg="white"
              placeholder="Identification"
              mb="10px"
              color="black"
              onChange={(e) => setIdInput(e.target.value)}
            />
            <Center>
              <Button type="submit" w="50%" colorScheme="teal" mb="0px">
                Issue Master Credential
              </Button>
            </Center>
          </form>
        </Flex>
        :
        <Flex pl="14%" pr="14%" direction="column" pb='40px' border='1px solid black' borderRadius='5px' mb='20px'>
          <Heading align="center" my="1rem" pb='20px'>Master Credential</Heading>
          <Text>Master Credential: {userMasterCredential}</Text>
        </Flex>
      }
      <Flex pl="14%" pr="14%" direction="column" pb='40px' border='1px solid black' borderRadius='5px' mb='20px'>
        <Heading align="center" my="1rem" pb='20px'>
            My Tickets
        </Heading>
        <Box border='1px solid black' borderRadius='5px' p='10px'>
          {eventName.length > 0 ? 
            <>
              <Heading size='lg'>Event Ticket</Heading>
              <Text>Event Name: {eventName}</Text>
              <Text>Number of Tickets Owned: {numTickets}</Text>
            </>
             : 
            <Text>No tickets</Text>
          }
          {numTickets.valueOf() !== "0".valueOf() && 
            <Button onClick={handleClickResellOpen} mt='10px'>Resell Tickets?</Button>
          }
          {resellTickets &&
            <>
              <Heading size='lg'>Resell Tickets</Heading>
              <form onSubmit={handleResellTickets}>
                <Input
                  bg="white"
                  placeholder="Number of Tickets to resell"
                  mb="10px"
                  color="black"
                  onChange={(e) => setResellNumTickets(e.target.value)}
                />
                <Input
                  bg="white"
                  placeholder="Resale Price per Ticket"
                  mb="10px"
                  color="black"
                  onChange={(e) => setResellPrice(e.target.value)}
                />
                <HStack>
                  <Button type="submit" w="50%" colorScheme="teal" mb="0px">
                    Resell Tickets
                  </Button>
                  <Button w="50%" colorScheme="red" mb="0px" onClick={handleClickResellClose}>
                    Cancel
                  </Button>
                </HStack>
              </form>
            </>
          }
          {numTickets.valueOf() !== "0".valueOf() && 
            <Button onClick={handleClickUseTicketsOpen} mt='10px' ml='10px'>Use Tickets?</Button>
          }
          {useTickets &&
            <>
              <Heading size='lg'>Use Tickets</Heading>
              <form onSubmit={handleUseTickets}>
                <Input
                  bg="white"
                  placeholder="Identification"
                  mb="10px"
                  color="black"
                  onChange={(e) => setUseTicketID(e.target.value)}
                />
                <Input
                  bg="white"
                  placeholder="Number of Tickets to use"
                  mb="10px"
                  color="black"
                  onChange={(e) => setUseTicketNumTickets(e.target.value)}
                />
                <HStack>
                  <Button type="submit" w="50%" colorScheme="teal" mb="0px">
                    Use Tickets
                  </Button>
                  <Button w="50%" colorScheme="red" mb="0px" onClick={handleClickUseTicketsClose}>
                    Cancel
                  </Button>
                </HStack>
              </form>
            </>
          }
        </Box>
      </Flex>
    </Flex>
  );
};

export default Homepage;