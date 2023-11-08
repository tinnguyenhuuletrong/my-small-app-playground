import useAppStore from "../stores/appStore";

export default function PanelNodeDetail() {
  const { activeNode } = useAppStore((state) => ({
    activeNode: state.nodes.find((itm) => itm.id === state.activeNode),
  }));

  if (!activeNode) return null;

  return (
    <div className="flex flex-col overflow-y-scroll bg-white py-6 shadow-xl w-80 min-h-screen">
      <div className="px-4 sm:px-6">
        <div className="flex items-start justify-between">
          <div className="text-base font-semibold leading-6 text-gray-900">
            Node Inspector
          </div>
        </div>
      </div>
      <hr />
      <div className="relative mt-6 flex-1 px-4 sm:px-6">
        {/* Your content */}
        <div className="flex flex-col gap-2">
          <div className="flex gap-7">
            <span>ID:</span>
            <span>{activeNode.id}</span>
          </div>
          <div className="flex gap-2">
            <span>Type:</span>
            <span>{activeNode.type}</span>
          </div>
          <div className="flex flex-col gap-2">
            <span>Data:</span>
            <pre className=" bg-slate-200 font-mono p-2 text-xs">
              {JSON.stringify(activeNode.data, null, " ")}
            </pre>
          </div>
        </div>
      </div>
    </div>
  );
}
