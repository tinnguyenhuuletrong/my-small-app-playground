export function classNames(...classes: string[]) {
  return classes.filter(Boolean).join(" ");
}

export const getNodeId = () => `randomnode_${+new Date()}`;
