import { FlowEditor } from "./components/FlowEditor";

function App() {
  return (
    <>
      <main className="w-screen h-screen">
        <div className=" bg-purple-50 flex flex-col items-center ">
          <div className="prose">
            <h1>This is title</h1>
          </div>
        </div>
        <FlowEditor />
      </main>
    </>
  );
}

export default App;
