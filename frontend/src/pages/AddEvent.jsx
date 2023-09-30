import React, { useEffect, useState, PureComponent } from "react";
import { useNavigate } from "react-router-dom";
import {
  Button,
  Heading,
  Flex,
  FormControl,
  FormLabel,
  FormErrorMessage,
  FormHelperText,
  Input,
  VStack,
} from "@chakra-ui/react";
import API from "../helpers/api";

const AddEvent = (field, form, ...props) => {
    const [name, setName] = React.useState("")
    const [numTickets, setNumTickets] = React.useState("")
    const [price, setPrice] = React.useState("")
    const [maxResalePrice, setMaxResalePrice] = React.useState("")
    const [resaleRoyalty, setResaleRoyalty] = React.useState("")

    const navigate = useNavigate();

    const [alertStatus, setAlertStatus] = useState("error");
    const [alertDisplay, setAlertDisplay] = useState("none");
    const [alertMessage, setAlertMessage] = useState("");

    const mbFormControl = '7px'
    const mbFormLabel = '3px'
  
    // Handle adding of event
    const handleAddEvent = (e) => {
      e.preventDefault();
      const details = {
        name: name,
        num_tickets: numTickets,
        price: price,
        max_resale_price: maxResalePrice,
        resale_royalty: resaleRoyalty,
      };
      console.log("HANDLE ADD Event")
      console.log(details)
      API.postAuthPath("sc/event/create", details)
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
          navigate("/home")
        });
    };

  return (
    <Flex w="100%" direction="column" align="center">
      <VStack border='1px solid black' borderRadius='5px' mt='10vh' p='30px' width='40%'>
        <Heading my="1rem">
          Add Event
        </Heading>
        <VStack>
          <form onSubmit={handleAddEvent}>
            <FormControl mb={mbFormControl}>
              <FormLabel mb={mbFormLabel}>Event Name</FormLabel>
              <Input type='text' onChange={(e) => setName(e.target.value)}/>
            </FormControl>
            <FormControl mb={mbFormControl}>
              <FormLabel mb={mbFormLabel}>Number of Tickets</FormLabel>
              <Input type='text' onChange={(e) => setNumTickets(e.target.value)}/>
            </FormControl>
            <FormControl mb={mbFormControl}>
              <FormLabel mb={mbFormLabel}>Price</FormLabel>
              <Input type='text' onChange={(e) => setPrice(e.target.value)}/>
            </FormControl>
            <FormControl mb={mbFormControl}>
              <FormLabel mb={mbFormLabel}>Max Resale Price</FormLabel>
              <Input type='text' onChange={(e) => setMaxResalePrice(e.target.value)}/>
            </FormControl>
            <FormControl mb={mbFormControl}>
              <FormLabel mb={mbFormLabel}>Resale Royalty</FormLabel>
              <Input type='text' onChange={(e) => setResaleRoyalty(e.target.value)}/>
            </FormControl>
            <Button type="submit" w="100%" colorScheme="teal" mb="15px" mt='20px'>
              Add Event
            </Button>
          </form>
        </VStack>
      </VStack>
    </Flex>
  );
};

export default AddEvent;