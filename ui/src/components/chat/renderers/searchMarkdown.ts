import type { Root, Element, Text, RootContent } from "hast";
import type { TextRange } from "../searchPassage";

// Runs after sanitization. It adds only owned text/mark nodes, never parsed
// HTML, and uses the renderer's original source offsets rather than searching
// rendered prose (which could select a repeated occurrence).
export function markMarkdownSource(tree: Root, source: string, ranges: TextRange[]) {
  const covered = new Set<TextRange>();
  const walk = (parent: Root | Element) => {
    const next: RootContent[] = [];
    for (const child of parent.children) {
      if (child.type === "element") walk(child);
      const start = child.position?.start.offset, end = child.position?.end.offset;
      if (child.type !== "text" || start === undefined || end === undefined || source.slice(start,end) !== child.value) { next.push(child); continue; }
      // Code fences have a dedicated renderer; preserve its plain child value.
      if (parent.type === "element" && parent.tagName === "code") { next.push(child); continue; }
      let cursor = 0;
      for (const range of ranges) {
        const a = Math.max(start,range.start), b = Math.min(end,range.end);
        if (b <= a) continue;
        const left = Math.max(cursor,a-start), right=b-start;
        if (right<=left) continue;
        next.push({type:"text",value:child.value.slice(cursor,left)} as Text);
        next.push({type:"element",tagName:"mark",properties:{"data-search-passage":"true"},children:[{type:"text",value:child.value.slice(left,right)}]} as Element);
        cursor=right;
        covered.add(range);
      }
      next.push({...child,value:child.value.slice(cursor)});
    }
    parent.children=next as typeof parent.children;
  };
  walk(tree);
  for (const range of ranges) {
    if (covered.has(range)) continue;
    let containing: Element | undefined;
    const find = (parent: Root | Element) => {
      for (const node of parent.children) {
        if (node.type!=="element") continue;
        const start=node.position?.start.offset,end=node.position?.end.offset;
        if (start!==undefined && end!==undefined && start<=range.start && end>=range.end) {
          containing=node;
          if (node.tagName!=="pre") find(node);
        }
      }
    };
    find(tree);
    if (containing) {
      // Syntax, image/link destinations and transformed code/diagram output
      // have no literal word position. Reveal their source block honestly.
      containing.properties["data-search-passage"]="true";
      const excerpt: Element={type:"element",tagName:"span",properties:{"data-search-excerpt":"true"},children:[{type:"text",value:`Matched source: ${source.slice(range.start,range.end)}`} ]};
      // Append beside the code node so its syntax renderer keeps string input.
      if (containing.tagName==="code") {
        tree.children.push(excerpt);
      } else containing.children.push(excerpt);
    }
  }
}
