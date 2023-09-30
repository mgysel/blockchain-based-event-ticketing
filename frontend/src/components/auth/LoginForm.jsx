import React, { useState, useContext } from "react";
import { useNavigate } from "react-router-dom";
import {
  Flex,
  Heading,
  Input,
  Button,
  InputGroup,
  InputRightElement,
  Alert,
  AlertTitle,
  AlertDescription,
  CloseButton,
  AlertIcon,
} from "@chakra-ui/react";
import { Link as RouterLink } from "react-router-dom";
import API from "../../helpers/api";
import { StoreContext } from "../../helpers/context";

const LoginForm = (props) => {
  const [show, setShow] = React.useState(false);
  const handleClick = () => setShow(!show);
  const [getEmail, setEmail] = useState("");
  const [getPassword, setPassword] = useState("");
  const navigate = useNavigate();
  const context = useContext(StoreContext);
  const setLoggedIn = context.loggedIn[1];
  const setPK = context.PK[1];
  const [alertStatus, setAlertStatus] = useState("error");
  const [alertDisplay, setAlertDisplay] = useState("none");
  const [alertMessage, setAlertMessage] = useState("");

  // Handle login of user
  const handleLogIn = (e) => {
    e.preventDefault();
    const details = {
      email: getEmail,
      password: getPassword,
    };
    API.postPath("auth/login", details)
      .then((json) => {
        localStorage.setItem("token", json["token"]);
        setLoggedIn(true);
        setPK(json["PK"]);
        navigate("/home");
      })
      .catch((err) => {
        err.json().then((json) => {
          setAlertDisplay("flex");
          setAlertStatus("error");
          setAlertMessage(json.message);
        });
      });
  };

  return (
    <Flex w="100%" p="1rem">
      <Flex
        boxShadow="lg"
        bg="gray.700"
        w="450px"
        borderRadius="10px"
        p="20px 60px 0px 60px"
        textAlign="center"
        color="white"
        direction="column"
        justify="space-between"
      >
        <Flex direction="column">
          <Heading pt="0px" pb="20px">
            Login
          </Heading>
          <form onSubmit={handleLogIn}>
            <Input
              bg="white"
              placeholder="Enter email"
              mb="10px"
              color="black"
              onChange={(e) => setEmail(e.target.value)}
            />
            <InputGroup
              size="md"
              bg="white"
              borderRadius="10px"
              mb="20px"
              color="black"
            >
              <Input
                pr="4.5rem"
                type={show ? "text" : "password"}
                placeholder="Enter password"
                onChange={(e) => setPassword(e.target.value)}
              />
              <InputRightElement width="4.5rem">
                <Button
                  h="1.75rem"
                  size="sm"
                  onClick={handleClick}
                  colorScheme="teal"
                  bg="gray.700"
                  color="white"
                >
                  {show ? "Hide" : "Show"}
                </Button>
              </InputRightElement>
            </InputGroup>
            <Button type="submit" w="100%" colorScheme="teal" mb="0px">
              Log in
            </Button>
          </form>
          <Alert
            status={alertStatus}
            my="0.5rem"
            display={alertDisplay}
            color="black"
          >
            <AlertIcon />
            <AlertTitle mr={2}>
              {alertStatus === "error" ? "Error" : "Success"}
            </AlertTitle>
            <AlertDescription>{alertMessage}</AlertDescription>
            <CloseButton
              onClick={() => {
                setAlertDisplay("none");
              }}
              position="absolute"
              right="8px"
              top="8px"
            />
          </Alert>
        </Flex>
        <Button
          color="white"
          variant="link"
          onClick={() => {props.setDisplay(1)}}
          mb='40px'
          mt='30px'
        >
          Don't have an account?
        </Button>
      </Flex>
    </Flex>
  );
};

export default LoginForm;
