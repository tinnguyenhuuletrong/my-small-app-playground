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
  NodeSelectionChange,
} from "reactflow";
import { DefaultNodes, DefaultEdges } from "./defaultData";

const SAVE_KEY = "my-rfs";

export type RFState = {
  nodes: Node[];
  edges: Edge[];
  activeNode?: string;
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
      // update active node
      const selectedEvent = changes.find(
        (itm) => itm.type === "select" && itm.selected === true
      );
      const unSelectedEvent = changes.find(
        (itm) => itm.type === "select" && itm.selected === false
      );
      let activeNode: string | undefined = get().activeNode;
      if (selectedEvent) {
        const tmp = selectedEvent as NodeSelectionChange;
        activeNode = tmp.id;
      } else if (unSelectedEvent) {
        activeNode = undefined;
      }

      set({
        nodes: applyNodeChanges(changes, get().nodes),
        activeNode,
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
