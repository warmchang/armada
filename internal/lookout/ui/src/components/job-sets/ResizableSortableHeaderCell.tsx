import React from "react"

import { ArrowDropDown, ArrowDropUp } from "@material-ui/icons"
import Draggable from "react-draggable"
import { TableHeaderProps } from "react-virtualized"
import "../SortableHeaderCell.css"

type ResizableSortableHeaderCellProps = {
  totalWidth: number
  currentState: Map<string, number>
  updateState: (state: Map<string, number>) => void
  columnId: string
  nextColumnId: string
  descending: boolean
  name: string
  className: string
  onOrderChange: (descending: boolean) => void
  onColumnResize: (
    deltaX: number,
    totalWidth: number,
    currentState: Map<string, number>,
    updateState: (state: Map<string, number>) => void,
    columnId: string,
    nextColumnId: string,
  ) => void
} & TableHeaderProps

export default function ResizableSortableHeaderCell(props: ResizableSortableHeaderCellProps) {
  return (
    <React.Fragment key={props.dataKey}>
      <div className={"sortable-header-cell " + props.className} onClick={() => props.onOrderChange(!props.descending)}>
        <div>{props.name}</div>
        <div>{props.descending ? <ArrowDropDown /> : <ArrowDropUp />}</div>
      </div>
      <Draggable
        axis="x"
        defaultClassName="DragHandle"
        defaultClassNameDragging="DragHandleActive"
        onDrag={(event, { deltaX }) => {
          props.onColumnResize(
            deltaX,
            props.totalWidth,
            props.currentState,
            props.updateState,
            props.columnId,
            props.nextColumnId,
          )
        }}
      >
        <span className="DragHandleIcon">⋮</span>
      </Draggable>
    </React.Fragment>
  )
}
