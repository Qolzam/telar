'use client';

import { useState } from 'react';
import { IconButton, Menu, MenuItem } from '@mui/material';
import { MoreVert, Edit, Delete } from '@mui/icons-material';

interface PostMenuProps {
  postId: string;
  onEdit: () => void;
  onDelete: () => void;
}

export function PostMenu({ postId, onEdit, onDelete }: PostMenuProps) {
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const open = Boolean(anchorEl);

  const handleClick = (event: React.MouseEvent<HTMLElement>) => {
    event.stopPropagation();
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  const handleEdit = () => {
    handleClose();
    onEdit();
  };

  const handleDelete = () => {
    handleClose();
    onDelete();
  };

  return (
    <>
      <IconButton
        aria-label="more options"
        onClick={handleClick}
        sx={{
          color: '#CBD5E1',
          '&:hover': { color: '#94A3B8' },
        }}
      >
        <MoreVert />
      </IconButton>
      <Menu
        anchorEl={anchorEl}
        open={open}
        onClose={handleClose}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
      >
        <MenuItem onClick={handleEdit}>
          <Edit sx={{ mr: 1, fontSize: 18 }} />
          Edit
        </MenuItem>
        <MenuItem onClick={handleDelete} sx={{ color: '#EF4444' }}>
          <Delete sx={{ mr: 1, fontSize: 18 }} />
          Delete
        </MenuItem>
      </Menu>
    </>
  );
}

