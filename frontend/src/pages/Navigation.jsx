import React from "react";
import { Routes, Route } from "react-router-dom";
import Landing from "./Landing";
import Homepage from "./Homepage";
import Events from "./Events";
import Event from "./Event";
import AddEvent from "./AddEvent";
import Admin from "./Admin";
import F3B from "./F3B";
import Value from "./Value";

const Navigation = () => {
  return (
    <Routes>
      <Route exact path="/" element={<Landing />}></Route>
      <Route exact path="/home" element={<Homepage />}></Route>
      <Route exact path="/events" element={<Events />}></Route>
      <Route exact path="/event/:event_id" element={<Event />}></Route>
      <Route exact path="/add-event" element={<AddEvent />}></Route>
      <Route exact path="/admin" element={<Admin />}></Route>
      <Route exact path="/f3b" element={<F3B />}></Route>
      <Route exact path="/value-contract" element={<Value />}></Route>
    </Routes>
  );
};

export default Navigation;
