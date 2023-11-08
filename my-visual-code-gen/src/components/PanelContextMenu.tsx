import { Node, useReactFlow } from "reactflow";
import ContextMenu from "./input/ContextMenu";
import useAppStore, { RFState } from "../stores/appStore";
import { getNodeId } from "../utils";

export type PanelContextMenuProps = {
  onClose?: () => void;
  top: number;
  left: number;
};

const selector = (state: RFState) => ({
  addNode: state.addNode,
});
export default function PanelContextMenu(props: PanelContextMenuProps) {
  const { addNode } = useAppStore(selector);
  const flowIns = useReactFlow();

  return (
    <>
      <ContextMenu
        options={[
          {
            label: "Add note node",
            onClick: () => {
              const pos = flowIns.project({
                x: props.left,
                y: props.top,
              });

              const newNode: Node = {
                id: getNodeId(),
                type: "noteNode",
                data: { value: "say something..." },
                position: pos,
              };
              addNode(newNode);

              props?.onClose && props.onClose();
            },
          },
          {
            label: "Add color node",
            onClick: () => {
              const pos = flowIns.project({
                x: props.left,
                y: props.top,
              });
              const newNode: Node = {
                id: getNodeId(),
                type: "colorNode",
                data: { value: "#91a8ee" },
                position: pos,
              };
              addNode(newNode);
              props?.onClose && props.onClose();
            },
          },
        ]}
      />
    </>
  );
}
