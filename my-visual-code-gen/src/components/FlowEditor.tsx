import { useState, useCallback } from "react";
import ReactFlow, {
  addEdge,
  FitViewOptions,
  applyNodeChanges,
  applyEdgeChanges,
  Node,
  Edge,
  DefaultEdgeOptions,
  OnConnect,
  OnEdgesChange,
  OnNodesChange,
  Background,
  Controls,
  MiniMap,
  BackgroundVariant,
  ReactFlowInstance,
  Panel,
} from "reactflow";
import "reactflow/dist/style.css";

const initialNodes: Node[] = [
  { id: "1", data: { label: "Node 1" }, position: { x: 5, y: 5 } },
  { id: "2", data: { label: "Node 2" }, position: { x: 5, y: 100 } },
];

const initialEdges: Edge[] = [{ id: "e1-2", source: "1", target: "2" }];

const getNodeId = () => `randomnode_${+new Date()}`;

const fitViewOptions: FitViewOptions = {
  padding: 0.2,
};

const defaultEdgeOptions: DefaultEdgeOptions = {
  animated: false,
};

export function FlowEditor() {
  const [nodes, setNodes] = useState<Node[]>(initialNodes);
  const [edges, setEdges] = useState<Edge[]>(initialEdges);
  const [flowIns, setFlowIns] = useState<ReactFlowInstance | null>(null);

  const onNodesChange: OnNodesChange = useCallback(
    (changes) => setNodes((nds) => applyNodeChanges(changes, nds)),
    [setNodes]
  );
  const onEdgesChange: OnEdgesChange = useCallback(
    (changes) => setEdges((eds) => applyEdgeChanges(changes, eds)),
    [setEdges]
  );
  const onConnect: OnConnect = useCallback(
    (connection) => setEdges((eds) => addEdge(connection, eds)),
    [setEdges]
  );

  const onSave = useCallback(() => {
    if (flowIns) {
      const flow = flowIns.toObject();
      console.log("saved: ", flow);
    }
  }, [flowIns]);

  const onAdd = useCallback(() => {
    if (!flowIns) return;

    const newNode = {
      id: getNodeId(),
      data: { label: "Added node" },
      position: {
        x: flowIns.getViewport().x,
        y: flowIns.getViewport().y,
      },
    };
    setNodes((nds) => nds.concat(newNode));
  }, [setNodes, flowIns]);

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      onConnect={onConnect}
      onInit={setFlowIns}
      fitView
      fitViewOptions={fitViewOptions}
      defaultEdgeOptions={defaultEdgeOptions}
    >
      <Panel position="top-right" className=" flex gap-2 ">
        <button className="btn btn-blue" onClick={onSave}>
          save
        </button>
        {/* <button onClick={onRestore}>restore</button> */}
        <button className="btn btn-blue" onClick={onAdd}>
          add node
        </button>
      </Panel>
      <Controls />
      <MiniMap />
      <Background variant={BackgroundVariant.Dots} gap={12} size={1} />
    </ReactFlow>
  );
}
