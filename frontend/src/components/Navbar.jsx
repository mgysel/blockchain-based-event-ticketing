import React, { useState, useContext, useEffect, useRef } from "react";
import { useNavigate } from "react-router-dom";
import {
  Flex,
  Heading,
  IconButton,
  Menu,
  MenuList,
  MenuItem,
  MenuButton,
  Text,
} from "@chakra-ui/react";
import { Link as RouterLink } from "react-router-dom";
import { FaUser } from "react-icons/fa";
import { StoreContext } from "../helpers/context";

const Navbar = () => {
  const context = useContext(StoreContext);
  const loggedIn = context.loggedIn[0];
  const setLoggedIn = context.loggedIn[1];

  const navigate = useNavigate();
  
  const navPadding = "5px";
  
  // Handle clicking logout
  const handleLogout = () => {
    console.log("Inside handleLogout")
    localStorage.removeItem("token");
    setLoggedIn(false);
    navigate("/");
  };

  return (
    <>
    { loggedIn && 
    <Flex h="3.5rem" justifyContent="center" bg="gray.700" color="white">
      <Flex
        w="100%"
        maxW="1366px"
        h="100%"
        alignItems="center"
        px="1rem"
        justifyContent="right"
      >
        <Text 
          as={RouterLink} to={"/home"}
          p={navPadding}
          onClick={() => {console.log("Clicking home")}}
        >
          Home
        </Text>
        <Text 
          as={RouterLink} to={"/events"}
          p={navPadding}
        >
          Events
        </Text>
        <Text 
          as={RouterLink} to={"/add-event"}
          p={navPadding}
        >
          Add Event
        </Text>
        <Text 
          as={RouterLink} to={"/admin"}
          p={navPadding}
        >
          Admin
        </Text>
        <Text 
          onClick={handleLogout}
          cursor='pointer'
          p={navPadding}
        >
          Logout
        </Text>
      </Flex>
    </Flex>
    }
    </>
  );
};

export default Navbar;
