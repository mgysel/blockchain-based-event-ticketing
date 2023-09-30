import React, { useEffect, useState, useContext } from "react";
import { useLocation, useNavigate, useParams } from "react-router-dom";
import {
  Box,
  Button,
  Center,
  Flex,
  Heading,
  HStack,
  Image,
  Input,
  Stack,
  Text,
  VStack,
} from "@chakra-ui/react";
import API from "../helpers/api";
import { StoreContext } from "../helpers/context";

const Event = (props) => {
  const context = useContext(StoreContext);

  const [name, setName] = React.useState("");
  const [numTickets, setNumTickets] = React.useState("");
  const [numTicketsLeft, setNumTicketsLeft] = React.useState("");
  const [price, setPrice] = React.useState("");
  const [maxResalePrice, setMaxResalePrice] = React.useState("");
  const [resaleRoyalty, setResaleRoyalty] = React.useState("");
  
  const [buyPrice, setBuyPrice] = React.useState("");
  const [buyNumTickets, setBuyNumTickets] = React.useState("");

  const [rebuyPrice, setRebuyPrice] = React.useState("");
  const [rebuyNumTickets, setRebuyNumTickets] = React.useState("");

  const location = useLocation();
  console.log("LOCATION");
  console.log(location);
  const navigate = useNavigate();

  // Styling
  const sidePadding='8%';
  const topPadding='40px';
  const boxPadding='20px';

  // Get all owned tickets
  useEffect(() => {
    const contractData = context.contractData[0];
    if (contractData != null) {
      setName(contractData.event_name);
      setNumTickets(contractData.num_tickets);
      setNumTicketsLeft(contractData.num_tickets_left);
      setPrice(contractData.price);
      setMaxResalePrice(contractData.max_resale_price);
      setResaleRoyalty(contractData.resale_royalty);
    } else {
      API.getPath("sc/event/read")
      .then((json) => {
        console.log("Getting event contract data")
        console.log(json)
        context.contractData[0] = json.data.contract_data;
        console.log("Contract data: ", context.contractData[0]);

        setName(contractData.event_name);
        setNumTickets(contractData.num_tickets);
        setNumTicketsLeft(contractData.num_tickets_left);
        setPrice(contractData.price);
        setMaxResalePrice(contractData.max_resale_price);
        setResaleRoyalty(contractData.resale_royalty);
      })
      .catch((err) => {
        console.warn(`Error: ${err}`);
      });
    }
  }, []);

  // Handle buy ticket for user
  const handleBuyTicket = (e) => {
    e.preventDefault();
    const details = {
      event_name: name,
      num_tickets: buyNumTickets,
      price: price,
    };
    API.postAuthPath("sc/event/buy", details)
      .then((json) => {
        console.log("Success");
        navigate("/home");
      })
      .catch((err) => {
        err.json().then((json) => {
          console.log("Error")
        });
      });
  };

  // Handle buy ticket for user
  const handleRebuyTicket = (e) => {
    e.preventDefault();
    const details = {
      event_name: name,
      num_tickets: rebuyNumTickets,
      price: price,
    };
    API.postAuthPath("sc/event/rebuy", details)
      .then((json) => {
        console.log("Success");
        console.log("Json: ", json)
        navigate("/home");
      })
      .catch((err) => {
        err.json().then((json) => {
          console.log("Error")
        });
      });
  };

  return (
    <>
      <VStack mt='40px' mb='20px' ml='8%' mr='8%'>
        {numTicketsLeft!=0 &&
          <VStack width='80vw' border='1px solid black' borderRadius='5px' p='20px' mb='20px'>
            <Heading>Primary Market Tickets</Heading>
            <HStack p='20px' mb='20px'>
              <Heading isTruncated fontWeight="semibold" pl='5px' pt='5px'>
                {name}
              </Heading>
              <Box border='3px solid black' borderRadius='10px' width='65vw' ml='10px' p='10px'>
                <Text>Number of Tickets Left: {numTicketsLeft}</Text>
                <Text>Price: {price}</Text>
                <Text>Max Resale Price: {maxResalePrice}</Text>
                <Text>Resale Royalty: {resaleRoyalty}</Text>
              </Box>
            </HStack>
            <Input
                bg="white"
                placeholder="Enter Number of Tickets"
                mb="10px"
                color="black"
                onChange={(e) => setBuyNumTickets(e.target.value)}
                width='300px'
              />
              <Button type="submit" w="100%" colorScheme="teal" mb="0px" width="300px" onClick={handleBuyTicket}>
                Buy Tickets
              </Button>
          </VStack>
        }
        {numTicketsLeft==0 &&
        <VStack width='80vw' border='1px solid black' borderRadius='5px' p='20px'>
          <Heading>Secondary Market Tickets</Heading>
          <Text>Tickets are sold out! But purchase some when they come available!</Text>
          <Input
            bg="white"
            placeholder="Enter Number of Tickets"
            mb="10px"
            color="black"
            onChange={(e) => setRebuyNumTickets(e.target.value)}
            width='300px'
          />
          <Input
            bg="white"
            placeholder="Enter Payment per Ticket"
            mb="10px"
            color="black"
            onChange={(e) => setRebuyPrice(e.target.value)}
            width='300px'
          />
          <Button type='submit' w="100%" colorScheme="teal" mb="0px" width="300px" onClick={handleRebuyTicket}>
            Buy Tickets
          </Button>
        </VStack>
        }
        </VStack>
    </>
  );
};

export default Event;