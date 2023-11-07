import { StateCreator, create } from "zustand";
import {
  Connection,
  Edge,
  EdgeChange,
  Node,
  NodeChange,
  addEdge,
  OnNodesChange,
  OnEdgesChange,
  OnConnect,
  applyNodeChanges,
  applyEdgeChanges,
  NodeAddChange,
} from "reactflow";
import { DefaultNodes, DefaultEdges } from "./defaultData";

const SAVE_KEY = "my-rfs";

export type RFState = {
  nodes: Node[];
  edges: Edge[];
  onNodesChange: OnNodesChange;
  onEdgesChange: OnEdgesChange;
  onConnect: OnConnect;
  addNode: (newNode: Node) => void;
  updateNodeData: (nodeId: string, data: Record<string, unknown>) => void;
  load: () => void;
  save: () => void;
  reset: () => void;
};

const myMiddlewareChain = (f: StateCreator<RFState>) => f;

// this is our useStore hook that we can use in our components to get parts of the store and call actions
const useAppStore = create<RFState>(
  myMiddlewareChain((set, get) => ({
    nodes: DefaultNodes,
    edges: DefaultEdges,

    reset() {
      window?.localStorage.removeItem(SAVE_KEY);
      set({
        nodes: DefaultNodes,
        edges: DefaultEdges,
      });
    },

    load() {
      const rawVal = window?.localStorage.getItem(SAVE_KEY);
      if (!rawVal) return;
      const saveData = JSON.parse(rawVal);

      set({
        nodes: saveData.nodes,
        edges: saveData.edges,
      });
    },

    save() {
      const { nodes, edges } = get();
      const saveData = { nodes, edges };
      console.log("saved: ", saveData);
      window?.localStorage.setItem(SAVE_KEY, JSON.stringify(saveData));
    },

    addNode: (newNode: Node) => {
      const addChange: NodeAddChange = {
        item: newNode,
        type: "add",
      };

      set({
        nodes: applyNodeChanges([addChange], get().nodes),
      });
    },

    updateNodeData: (nodeId: string, data: Record<string, unknown>) => {
      const nodeIndex = get().nodes.findIndex((itm) => itm.id === nodeId);
      if (nodeIndex < 0) return;

      const originalData = get().nodes;
      const newData = [...originalData];
      newData[nodeIndex] = { ...originalData[nodeIndex], data };

      set({
        nodes: newData,
      });
    },

    onNodesChange: (changes: NodeChange[]) => {
      set({
        nodes: applyNodeChanges(changes, get().nodes),
      });
    },
    onEdgesChange: (changes: EdgeChange[]) => {
      set({
        edges: applyEdgeChanges(changes, get().edges),
      });
    },
    onConnect: (connection: Connection) => {
      set({
        edges: addEdge(connection, get().edges),
      });
    },
  }))
);

export default useAppStore;
