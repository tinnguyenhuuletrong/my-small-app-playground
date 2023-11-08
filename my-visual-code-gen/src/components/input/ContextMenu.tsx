export type ContextMenuProps = {
  options: Array<{
    label: string;
    onClick?: () => void;
  }>;
};

export default function ContextMenu(props: ContextMenuProps) {
  return (
    <div className="right-0 w-56 origin-top-right rounded-md bg-white shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none">
      <div className="py-1 flex flex-col gap-2">
        {props.options.map((itm) => (
          <span
            className="px-2 py-2 hover:bg-slate-100 cursor-pointer"
            onClick={itm.onClick}
          >
            {itm.label}
          </span>
        ))}
      </div>
    </div>
  );
}
