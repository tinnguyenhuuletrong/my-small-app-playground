import { useCallback } from "react";
import Dropdown from "./input/Dropdown";
import useAppStore, { RFState } from "../stores/appStore";

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
  const { save, load, reset } = useAppStore(selector);

  const onSave = useCallback(() => {
    save();
  }, [save]);

  const onLoad = useCallback(() => {
    load();
  }, [load]);

  const onReset = useCallback(() => {
    reset();
  }, [reset]);

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
