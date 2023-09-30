import React from "react";

export const StoreContext = React.createContext(null);

// useContext for navbar
function StoreProvider({ children }) {
  const [loggedIn, setLoggedIn] = React.useState(
    localStorage.getItem("token") ? true : false
  );
  const [contractData, setContractData] = React.useState(null);
  const [PK, setPK] = React.useState("");
  const [masterCredential, setMasterCredential] = React.useState("");
  const [idHash, setIdHash] = React.useState("");

  const store = {
    loggedIn: [loggedIn, setLoggedIn],
    contractData: [contractData, setContractData],
    PK: [PK, setPK],
    masterCredential: [masterCredential, setMasterCredential],
    idHash: [idHash, setIdHash],
  };

  return (
    <StoreContext.Provider value={store}>{children}</StoreContext.Provider>
  );
}

export default StoreProvider;
