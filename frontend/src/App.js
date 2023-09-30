import Navbar from "./components/Navbar";
import Navigation from "./pages/Navigation";
import StoreProvider from "./helpers/context";

function App() {
  return (
    <div className="App">
      <StoreProvider>
        <Navbar />
        <Navigation />
      </StoreProvider>
    </div>
  );
}

export default App;
