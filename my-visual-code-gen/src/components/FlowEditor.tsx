import { useState, useCallback } from "react";
import ReactFlow, {
  FitViewOptions,
  DefaultEdgeOptions,
  Background,
  Controls,
  MiniMap,
  BackgroundVariant,
  ReactFlowInstance,
  Panel,
  NodeTypes,
  Node,
} from "reactflow";
import "reactflow/dist/style.css";

import useAppStore, { RFState } from "../stores/appStore";
import NoteCustomNode from "./NoteCustomNode";
import Dropdown from "./input/Dropdown";
import { getNodeId } from "../utils";

const fitViewOptions: FitViewOptions = {
  padding: 0.2,
};

const defaultEdgeOptions: DefaultEdgeOptions = {
  animated: false,
};

const nodeTypes: NodeTypes = {
  noteNode: NoteCustomNode,
};

const selector = (state: RFState) => ({
  nodes: state.nodes,
  edges: state.edges,
  onNodesChange: state.onNodesChange,
  onEdgesChange: state.onEdgesChange,
  onConnect: state.onConnect,
  addNode: state.addNode,
  save: state.save,
  load: state.load,
  reset: state.reset,
});

export function FlowEditor() {
  const {
    nodes,
    edges,
    onNodesChange,
    onEdgesChange,
    onConnect,
    addNode,
    save,
    load,
    reset,
  } = useAppStore(selector);
  const [flowIns, setFlowIns] = useState<ReactFlowInstance | null>(null);

  const onSave = useCallback(() => {
    save();
  }, [save]);

  const onLoad = useCallback(() => {
    load();
  }, [load]);

  const onReset = useCallback(() => {
    reset();
  }, [reset]);

  const onAdd = useCallback(() => {
    if (!flowIns) return;

    const newNode: Node = {
      id: getNodeId(),
      type: "noteNode",
      data: { value: "say something..." },
      position: {
        x: flowIns.getViewport().x,
        y: flowIns.getViewport().y,
      },
    };

    addNode(newNode);
  }, [addNode, flowIns]);

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
      nodeTypes={nodeTypes}
    >
      <Panel position="top-right" className=" flex gap-2 ">
        <button className="btn btn-blue" onClick={onSave}>
          save
        </button>
        <button className="btn btn-blue" onClick={onLoad}>
          load
        </button>
        <button className="btn btn-yellow" onClick={onReset}>
          reset
        </button>
        <button className="btn btn-blue" onClick={onAdd}>
          add node
        </button>

        <Dropdown
          classNames="btn btn-blue"
          label="Pick one"
          options={[
            {
              label: "Option 1",
              onClick: () => console.log("on option 1 choice"),
            },
            {
              label: "Option 2",
              onClick: () => console.log("on option 2 choice"),
            },
          ]}
        />
      </Panel>
      <Controls />
      <MiniMap />
      <Background variant={BackgroundVariant.Dots} gap={12} size={1} />
    </ReactFlow>
  );
}
