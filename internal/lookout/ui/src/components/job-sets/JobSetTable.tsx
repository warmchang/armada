import React from "react"

import Draggable from "react-draggable"
import { TableCellProps, Table } from "react-virtualized"
import { Column, defaultTableCellRenderer } from "react-virtualized"
import { TableHeaderProps } from "react-virtualized/dist/es/Table"

import { JobSet } from "../../services/JobService"
import CheckboxHeaderRow from "../CheckboxHeaderRow"
import CheckboxRow from "../CheckboxRow"
import LinkCell from "../LinkCell"
import SortableHeaderCell from "../SortableHeaderCell"
import "./JobSetTable.css"
import ResizableSortableHeaderCell from "./ResizableSortableHeaderCell"

const jobSetColumnId = "jobSetId"
const submissionTimeColumnId = "submissionTime"
const jobsQueuedColumnId = "jobsQueued"
const jobsPendingColumnId = "jobsPending"
const jobsRunningColumnId = "jobsRunning"
const jobsSucceededColumnId = "jobsSucceeded"
const jobsFailedColumnId = "jobsFailed"

export const defaultColumnWeights = new Map<string, number>([
  [jobSetColumnId, 0.35],
  [submissionTimeColumnId, 0.15],
  [jobsQueuedColumnId, 0.1],
  [jobsPendingColumnId, 0.1],
  [jobsRunningColumnId, 0.1],
  [jobsSucceededColumnId, 0.1],
  [jobsFailedColumnId, 0.1],
])

export interface JobSetTableProps {
  height: number
  width: number
  jobSets: JobSet[]
  selectedJobSets: Map<string, JobSet>
  newestFirst: boolean
  onJobSetClick: (jobSet: string, state: string) => void
  onSelectJobSet: (index: number, selected: boolean) => void
  onShiftSelectJobSet: (index: number, selected: boolean) => void
  onDeselectAllClick: () => void
  onSelectAllClick: () => void
  onOrderChange: (newestFirst: boolean) => void
  columnWeights: Map<string, number>
  onColumnWeightUpdate: (columnWeights: Map<string, number>) => void
}

function headerRenderer(
  tableHeaderProps: TableHeaderProps,
  tableProps: JobSetTableProps,
  columnId: string,
  nextColumnId: string,
) {
  return (
    <React.Fragment key={tableHeaderProps.dataKey}>
      <div className="ReactVirtualized__Table__headerTruncatedText">{tableHeaderProps.label}</div>
      <Draggable
        axis="x"
        defaultClassName="DragHandle"
        defaultClassNameDragging="DragHandleActive"
        onDrag={(event, { deltaX }) => {
          onColumnDrag(
            deltaX,
            tableProps.width,
            tableProps.columnWeights,
            tableProps.onColumnWeightUpdate,
            columnId,
            nextColumnId,
          )
        }}
      >
        <span className="DragHandleIcon">⋮</span>
      </Draggable>
    </React.Fragment>
  )
}

function onColumnDrag(
  deltaX: number,
  totalWidth: number,
  currentState: Map<string, number>,
  updateState: (state: Map<string, number>) => void,
  columnId: string,
  nextColumnId: string,
) {
  if (currentState.has(columnId) && currentState.has(nextColumnId)) {
    const newValue = new Map<string, number>(currentState)
    const percentDelta = deltaX / totalWidth
    newValue.set(columnId, currentState.get(columnId)! + percentDelta)
    newValue.set(nextColumnId, currentState.get(nextColumnId)! - percentDelta)
    updateState(newValue)
  } else {
    console.error("Not updating columns as could not find columns being altered")
  }
}

function cellRendererForState(
  cellProps: TableCellProps,
  onJobSetClick: (jobSet: string, state: string) => void,
  state: string,
) {
  if (cellProps.cellData) {
    return <LinkCell onClick={() => onJobSetClick((cellProps.rowData as JobSet).jobSetId, state)} {...cellProps} />
  }
  return defaultTableCellRenderer(cellProps)
}

export default function JobSetTable(props: JobSetTableProps) {
  console.log("Render called")
  return (
    <div
      style={{
        height: props.height,
        width: props.width,
      }}
    >
      <Table
        rowGetter={({ index }) => props.jobSets[index]}
        rowCount={props.jobSets.length}
        rowHeight={40}
        headerHeight={60}
        height={props.height}
        width={props.width}
        headerClassName="job-set-table-header"
        rowRenderer={(tableRowProps) => {
          return (
            <CheckboxRow
              isChecked={props.selectedJobSets.has(tableRowProps.rowData.jobSetId)}
              onChangeChecked={(selected) => props.onSelectJobSet(tableRowProps.index, selected)}
              onChangeCheckedShift={(selected) => props.onShiftSelectJobSet(tableRowProps.index, selected)}
              tableKey={tableRowProps.key}
              {...tableRowProps}
            />
          )
        }}
        headerRowRenderer={(tableHeaderRowProps) => {
          const jobSetsAreSelected = props.selectedJobSets.size > 0
          const noJobSetsArePresent = props.jobSets.length == 0
          return (
            <CheckboxHeaderRow
              checked={jobSetsAreSelected}
              disabled={!jobSetsAreSelected && noJobSetsArePresent}
              onClick={jobSetsAreSelected ? () => props.onDeselectAllClick() : props.onSelectAllClick}
              {...tableHeaderRowProps}
            />
          )
        }}
      >
        <Column
          dataKey="jobSetId"
          className="job-set-table-job-set-name-cell"
          headerRenderer={(headerProps) => headerRenderer(headerProps, props, jobSetColumnId, submissionTimeColumnId)}
          width={props.columnWeights.get(jobSetColumnId)! * props.width}
          label="Job Set"
        />
        <Column
          dataKey="latestSubmissionTime"
          width={props.columnWeights.get(submissionTimeColumnId)! * props.width}
          label="Submission Time"
          headerRenderer={(cellProps) => (
            <ResizableSortableHeaderCell
              totalWidth={props.width}
              currentState={props.columnWeights}
              updateState={props.onColumnWeightUpdate}
              columnId={submissionTimeColumnId}
              nextColumnId={jobsQueuedColumnId}
              name="Submission Time"
              descending={props.newestFirst}
              className="job-set-submission-time-header-cell"
              onOrderChange={props.onOrderChange}
              onColumnResize={onColumnDrag}
              {...cellProps}
            />
          )}
        />
        <Column
          dataKey="jobsQueued"
          width={props.columnWeights.get(jobsQueuedColumnId)! * props.width}
          label="Queued"
          className="job-set-table-number-cell"
          headerRenderer={(headerProps) => headerRenderer(headerProps, props, jobsQueuedColumnId, jobsPendingColumnId)}
          cellRenderer={(cellProps) => cellRendererForState(cellProps, props.onJobSetClick, "Queued")}
        />
        <Column
          dataKey="jobsPending"
          width={props.columnWeights.get(jobsPendingColumnId)! * props.width}
          label="Pending"
          className="job-set-table-number-cell"
          headerRenderer={(headerProps) => headerRenderer(headerProps, props, jobsPendingColumnId, jobsRunningColumnId)}
          cellRenderer={(cellProps) => cellRendererForState(cellProps, props.onJobSetClick, "Pending")}
        />
        <Column
          dataKey="jobsRunning"
          width={props.columnWeights.get(jobsRunningColumnId)! * props.width}
          label="Running"
          className="job-set-table-number-cell"
          headerRenderer={(headerProps) =>
            headerRenderer(headerProps, props, jobsRunningColumnId, jobsSucceededColumnId)
          }
          cellRenderer={(cellProps) => cellRendererForState(cellProps, props.onJobSetClick, "Running")}
        />
        <Column
          dataKey="jobsSucceeded"
          width={props.columnWeights.get(jobsSucceededColumnId)! * props.width}
          label="Succeeded"
          className="job-set-table-number-cell"
          headerRenderer={(headerProps) =>
            headerRenderer(headerProps, props, jobsSucceededColumnId, jobsFailedColumnId)
          }
          cellRenderer={(cellProps) => cellRendererForState(cellProps, props.onJobSetClick, "Succeeded")}
        />
        <Column
          dataKey="jobsFailed"
          width={props.columnWeights.get(jobsFailedColumnId)! * props.width}
          label="Failed"
          className="job-set-table-number-cell"
          cellRenderer={(cellProps) => cellRendererForState(cellProps, props.onJobSetClick, "Failed")}
        />
      </Table>
    </div>
  )
}
