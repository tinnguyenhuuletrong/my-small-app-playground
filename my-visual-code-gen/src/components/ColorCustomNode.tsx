import { Handle, NodeProps, Position } from "reactflow";
import useAppStore from "../stores/appStore";

type NodeData = {
  value: string;
};

export default function ColorCustomNode({ id, data }: NodeProps<NodeData>) {
  const updateNodeData = useAppStore((state) => state.updateNodeData);

  return (
    <>
      <div
        className="flex rounded-sm "
        style={{
          backgroundColor: data.value,
        }}
      >
        <input
          type="color"
          value={data.value}
          className="m-1 p-1"
          onChange={(e) => {
            updateNodeData(id, { value: e.target.value });
          }}
        />
      </div>
      <Handle type="source" position={Position.Bottom} />
    </>
  );
}
