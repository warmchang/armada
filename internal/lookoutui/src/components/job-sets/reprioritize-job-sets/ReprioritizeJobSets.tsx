import { Button, List, ListItem, ListItemText, Paper, TextField } from "@mui/material"

import { JobSet } from "../../../services/JobService"

import "./ReprioritizeJobSets.css"
import "../../Dialog.css"
import "../../Text.css"

type ReprioritizeJobSetsProps = {
  queue: string
  jobSets: JobSet[]
  isLoading: boolean
  isValid: boolean
  onReprioritizeJobsSets: () => void
  onPriorityChange: (priority: string) => void
}

export default function ReprioritizeJobSets(props: ReprioritizeJobSetsProps) {
  return (
    <div className="lookout-dialog-container">
      <p className="lookout-dialog-fixed">The following Job Sets in queue {props.queue} will be reprioritized:</p>
      <List component={Paper} className="lookout-dialog-varying reprioritize-job-sets">
        {props.jobSets.map((jobSet) => (
          <ListItem key={jobSet.jobSetId}>
            <ListItemText className="lookout-word-wrapped">{jobSet.jobSetId}</ListItemText>
          </ListItem>
        ))}
      </List>
      <div className="lookout-dialog-centered lookout-dialog-fixed reprioritize-job-sets-options">
        <div>
          <TextField
            autoFocus={true}
            placeholder={"New priority"}
            type={"text"}
            error={!props.isValid}
            helperText={!props.isValid ? "Value must be a number >= 0" : " "}
            onChange={(event) => props.onPriorityChange(event.target.value)}
          />
        </div>
        <div>
          <Button disabled={!props.isValid} loading={props.isLoading} onClick={props.onReprioritizeJobsSets}>
            Reprioritize Job Sets
          </Button>
        </div>
      </div>
    </div>
  )
}
