import ReactFlow, {
  FitViewOptions,
  DefaultEdgeOptions,
  Background,
  Controls,
  MiniMap,
  BackgroundVariant,
  Panel,
  NodeTypes,
} from "reactflow";
import "reactflow/dist/style.css";

import useAppStore, { RFState } from "../stores/appStore";
import NoteCustomNode from "./NoteCustomNode";
import PanelNodeDetail from "./PanelNodeDetail";
import PanelTopMenu from "./PanelTopMenu";
import { useState } from "react";
import PanelContextMenu from "./PanelContextMenu";
import ColorCustomNode from "./ColorCustomNode";

const fitViewOptions: FitViewOptions = {
  padding: 0.2,
};

const defaultEdgeOptions: DefaultEdgeOptions = {
  animated: false,
};

const nodeTypes: NodeTypes = {
  noteNode: NoteCustomNode,
  colorNode: ColorCustomNode,
};

const selector = (state: RFState) => ({
  nodes: state.nodes,
  edges: state.edges,
  onNodesChange: state.onNodesChange,
  onEdgesChange: state.onEdgesChange,
  onConnect: state.onConnect,
});

export function FlowEditor() {
  const { nodes, edges, onNodesChange, onEdgesChange, onConnect } =
    useAppStore(selector);
  const [contextMenu, setContextMenu] = useState<{
    top: number;
    left: number;
  } | null>(null);

  return (
    <ReactFlow
      nodes={nodes}
      edges={edges}
      onNodesChange={onNodesChange}
      onEdgesChange={onEdgesChange}
      onConnect={onConnect}
      onPaneClick={() => {
        setContextMenu(null);
      }}
      onPaneContextMenu={(e) => {
        e.preventDefault();

        const contextMenuParams = {
          top: e.clientY - 50,
          left: e.clientX,
        };
        setContextMenu(contextMenuParams);
      }}
      fitView
      fitViewOptions={fitViewOptions}
      defaultEdgeOptions={defaultEdgeOptions}
      nodeTypes={nodeTypes}
    >
      <Panel position="top-right" className=" flex gap-2 ">
        <PanelTopMenu />
      </Panel>
      <Panel position="top-left" className="m-0">
        <PanelNodeDetail />
      </Panel>

      <Controls />
      <MiniMap />
      <Background variant={BackgroundVariant.Dots} gap={12} size={1} />

      {contextMenu && (
        <>
          <div
            style={{ top: contextMenu.top, left: contextMenu.left }}
            className="absolute z-10"
          >
            <PanelContextMenu
              {...contextMenu}
              onClose={() => {
                setContextMenu(null);
              }}
            />
          </div>
        </>
      )}
    </ReactFlow>
  );
}
