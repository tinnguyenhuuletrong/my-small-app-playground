import { Handle, NodeProps, Position } from "reactflow";
import useAppStore from "../stores/appStore";

type NodeData = {
  value: string;
};

export default function NoteCustomNode({ id, data }: NodeProps<NodeData>) {
  const updateNodeData = useAppStore((state) => state.updateNodeData);

  return (
    <>
      <Handle type="target" position={Position.Top} />
      <div className="flex bg-slate-200 rounded-sm ">
        <textarea
          value={data.value}
          onChange={(e) => {
            updateNodeData(id, { value: e.target.value });
          }}
          className="m-1 p-1"
        />
      </div>
      <Handle type="source" position={Position.Bottom} />
    </>
  );
}
