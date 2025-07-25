import { Job } from "./lookoutModels"
import { RowId } from "../utils/reactTableUtils"

export interface BaseJobTableRow {
  rowId: RowId
}

export type JobRow = BaseJobTableRow & Partial<Job>
export type JobGroupRow = JobRow & {
  isGroup: true // The ReactTable version of this doesn't seem to play nice with manual/serverside expanding
  jobCount?: number
  subRows: JobTableRow[]
  groupedField: string
  stateCounts: Record<string, number> | undefined
}

export type JobTableRow = JobRow | JobGroupRow

export const isJobGroupRow = (row?: JobTableRow): row is JobGroupRow => row !== undefined && "isGroup" in row
