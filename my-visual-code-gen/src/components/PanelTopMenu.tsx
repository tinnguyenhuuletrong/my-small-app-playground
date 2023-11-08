import { useCallback } from "react";
import { getNodeId } from "../utils";
import Dropdown from "./input/Dropdown";
import useAppStore, { RFState } from "../stores/appStore";
import { useReactFlow, Node } from "reactflow";

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

export default function PanelTopMenu() {
  const flowIns = useReactFlow();
  const { addNode, save, load, reset } = useAppStore(selector);

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
    <>
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
    </>
  );
}
