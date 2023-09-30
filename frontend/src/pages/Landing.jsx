import React, { useEffect, useState, useContext } from "react";
import { Flex, Heading, Button } from "@chakra-ui/react";
import { Link as RouterLink } from "react-router-dom";
import LoginForm from "../components/auth/LoginForm.jsx";
import SignupForm from "../components/auth/SignupForm.jsx";

const Landing = () => {
  const [display, setDisplay] = React.useState(0);

  useEffect(() => {
    document.body.style.overflow = 'hidden'

  }, [])

  return (
    <div style={{ 
      backgroundImage: `url(${process.env.PUBLIC_URL + 'landing_background.jpg'})`, 
      backgroundRepeat: 'no-repeat',
      backgroundSize: 'cover',
      width: '100vw',
      height: '100vh'
    }}>
      <Flex
        w="100%"
        maxW="1366px"
        p="1rem"
        flexDirection="column"
        alignItems="center"
      >
      <Heading as="h1" my="1rem" fontSize="4em" mt="10vh">
        Welcome to NFTickets
      </Heading>
      <Flex justifyContent="space-between" w="100%" maxW="28rem" mt="4vh">
        {
          display===0 &&
          <>
            <LoginForm setDisplay={setDisplay} />
          </>
        }
        {
          display===1 &&
          <>
            <SignupForm setDisplay={setDisplay} />
          </>
        }
      </Flex>
    </Flex>
    </div>
  );
};

export default Landing;
