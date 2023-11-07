import { Edge, Node } from "reactflow";

export const DefaultNodes = [
  {
    id: "1",
    type: "input",
    data: { label: "Input" },
    position: { x: 250, y: 25 },
  },
  {
    id: "3",
    type: "output",
    data: { label: "Output" },
    position: { x: 250, y: 250 },
  },
] as Node[];

export const DefaultEdges = [] as Edge[];
