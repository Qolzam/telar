'use client';

import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
} from '@mui/material';

interface ConfirmDeleteDialogProps {
  open: boolean;
  onCancel: () => void;
  onConfirm: () => void;
}

export function ConfirmDeleteDialog({ open, onCancel, onConfirm }: ConfirmDeleteDialogProps) {
  return (
    <Dialog open={open} onClose={onCancel} aria-labelledby="confirm-delete-title">
      <DialogTitle id="confirm-delete-title">Delete comment?</DialogTitle>
      <DialogContent>
        <DialogContentText>
          This action cannot be undone.
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={onCancel} variant="text">Cancel</Button>
        <Button onClick={onConfirm} color="error" variant="contained" autoFocus>
          Delete
        </Button>
      </DialogActions>
    </Dialog>
  );
}






