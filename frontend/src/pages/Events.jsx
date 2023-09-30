import React, { useEffect, useState } from "react";
import { useLocation, Link as RouterLink } from "react-router-dom";

import {
  Heading,
  Flex,
  Grid,
  GridItem,
  LinkBox,
  LinkOverlay,
  Text,
  Wrap,
  WrapItem
} from "@chakra-ui/react";
import API from "../helpers/api";

const Events = () => {
  const [events, setEvents] = useState([]);

  // Get user profile
  useEffect(() => {
    API.getPath("sc/event/get-events")
      .then((json) => {
        setEvents(json.data);
      })
      .catch((err) => {
        console.warn(`Error: ${err}`);
      });
  }, []);

  return (
    <Flex pl="14%" pr="14%" direction="column" pb='40px'>
        <Heading align="center" my="1rem" pb='20px'>
            Events
        </Heading>
        <LinkBox maxW='80vw' p='5'>
          <Grid templateColumns='repeat(2, 3fr)' gap={10}>
            {events.map((id, index) => (
              <LinkOverlay key={index} as={RouterLink} to={`/event/${id._id}`} state={{"eventID": id._id}} border='1px solid black' borderRadius='5px' p='10px'>
                <GridItem>
                  <Heading>Event Name: {id.name}</Heading>
                  <Text>Number of tickets: {id.num_tickets}</Text>
                  <Text>Ticket price: ${id.price}</Text>
                  <Text>Maximum resale price: ${id.max_resale_price}</Text>
                  <Text>Resale royalty to event organizer: {id.resale_royalty}%</Text>
                </GridItem>
              </LinkOverlay>
              ))}
          </Grid>
        </LinkBox>
    </Flex>
  );
};

export default Events;