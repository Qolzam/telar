'use client';

import { useEffect, useMemo, useState, type ChangeEvent } from 'react';
import { Box, Button, CircularProgress, Dialog, DialogActions, DialogContent, DialogContentText, DialogTitle, MenuItem, Paper, Select, Snackbar, Table, TableBody, TableCell, TableContainer, TableHead, TableRow, TextField, Typography, IconButton } from '@mui/material';
import type { AdminMember } from '@telar/sdk';
import type { SelectChangeEvent } from '@mui/material/Select';
import { useMembersQuery, useUpdateMemberRoleMutation, useBanUserMutation } from '../client';
import ArrowUpwardIcon from '@mui/icons-material/ArrowUpward';
import ArrowDownwardIcon from '@mui/icons-material/ArrowDownward';

export function MembersTable() {
  const [limit, setLimit] = useState(20);
  const [offset, setOffset] = useState(0);
  const [sortBy, setSortBy] = useState<'created_date' | 'full_name' | 'email'>('created_date');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');
  const [searchText, setSearchText] = useState('');
  const [debouncedSearch, setDebouncedSearch] = useState('');

  useEffect(() => {
    const id = setTimeout(() => setDebouncedSearch(searchText), 500);
    return () => clearTimeout(id);
  }, [searchText]);

  const { data, isLoading, isError } = useMembersQuery({ limit, offset, search: debouncedSearch, sortBy, sortOrder });
  const updateRole = useUpdateMemberRoleMutation();
  const banUser = useBanUserMutation();
  const [snack, setSnack] = useState<{ open: boolean; message: string }>({ open: false, message: '' });
  const [confirm, setConfirm] = useState<{ open: boolean; userId?: string }>({ open: false });

  const rows = useMemo<AdminMember[]>(() => data?.members ?? [], [data]);

  const toggleSort = (field: 'created_date' | 'full_name' | 'email') => {
    if (sortBy === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortBy(field);
      setSortOrder('asc');
    }
  };

  if (isLoading) {
    return (
      <Box sx={{ py: 6, display: 'flex', justifyContent: 'center' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (isError) {
    return (
      <Typography variant="body2" color="error">
        Failed to load members.
      </Typography>
    );
  }

  return (
    <TableContainer component={Paper}>
      <Box sx={{ p: 2, display: 'flex', gap: 2, alignItems: 'center' }}>
        <TextField
          size="small"
          placeholder="Search members…"
          value={searchText}
          onChange={(e: ChangeEvent<HTMLInputElement>) => {
            setOffset(0);
            setSearchText(e.target.value);
          }}
          sx={{ width: 320 }}
        />
      </Box>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                Name
                <IconButton size="small" onClick={() => toggleSort('full_name')}>
                  {sortBy === 'full_name' && sortOrder === 'asc' ? <ArrowUpwardIcon fontSize="inherit" /> : <ArrowDownwardIcon fontSize="inherit" />}
                </IconButton>
              </Box>
            </TableCell>
            <TableCell>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                Email
                <IconButton size="small" onClick={() => toggleSort('email')}>
                  {sortBy === 'email' && sortOrder === 'asc' ? <ArrowUpwardIcon fontSize="inherit" /> : <ArrowDownwardIcon fontSize="inherit" />}
                </IconButton>
              </Box>
            </TableCell>
            <TableCell>Role</TableCell>
            <TableCell>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                Joined
                <IconButton size="small" onClick={() => toggleSort('created_date')}>
                  {sortBy === 'created_date' && sortOrder === 'asc' ? <ArrowUpwardIcon fontSize="inherit" /> : <ArrowDownwardIcon fontSize="inherit" />}
                </IconButton>
              </Box>
            </TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {rows.map((m: AdminMember) => (
            <TableRow key={m.objectId}>
              <TableCell>{m.displayName}</TableCell>
              <TableCell>{m.email}</TableCell>
              <TableCell>
                <Select
                  size="small"
                  value={m.role}
                  onChange={(e: SelectChangeEvent<string>) =>
                    updateRole.mutate(
                      { userId: m.objectId, role: String(e.target.value) },
                      {
                        onSuccess: () => setSnack({ open: true, message: 'User role updated.' }),
                        onError: () => setSnack({ open: true, message: 'Failed to update role.' }),
                      }
                    )
                  }
                >
                  <MenuItem value="user">User</MenuItem>
                  <MenuItem value="moderator">Moderator</MenuItem>
                  <MenuItem value="admin">Admin</MenuItem>
                </Select>
              </TableCell>
              <TableCell>
                {new Date(m.createdDate * 1000).toLocaleDateString()}
                <Button
                  size="small"
                  color="error"
                  sx={{ ml: 2 }}
                  onClick={() => setConfirm({ open: true, userId: m.objectId })}
                >
                  Ban
                </Button>
              </TableCell>
            </TableRow>
          ))}
          {rows.length === 0 && (
            <TableRow>
              <TableCell colSpan={4}>
                <Typography variant="body2" color="text.secondary">
                  No members found.
                </Typography>
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', p: 2 }}>
        <Typography variant="caption">
          Showing {rows.length} of {data?.total ?? 0}
        </Typography>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <IconButton
            size="small"
            disabled={offset <= 0}
            onClick={() => setOffset(Math.max(0, offset - limit))}
          >
            <Typography variant="button">Prev</Typography>
          </IconButton>
          <IconButton
            size="small"
            disabled={!!data && offset + limit >= (data.total ?? 0)}
            onClick={() => setOffset(offset + limit)}
          >
            <Typography variant="button">Next</Typography>
          </IconButton>
        </Box>
      </Box>
      <Snackbar
        open={snack.open}
        onClose={() => setSnack({ ...snack, open: false })}
        message={snack.message}
        autoHideDuration={3000}
      />
      <Dialog open={confirm.open} onClose={() => setConfirm({ open: false })}>
        <DialogTitle>Ban user?</DialogTitle>
        <DialogContent>
          <DialogContentText>
            This action will ban the user and may revoke sessions. Are you sure you want to continue?
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirm({ open: false })}>Cancel</Button>
          <Button
            color="error"
            onClick={() => {
              if (confirm.userId) {
                banUser.mutate(confirm.userId, {
                  onSuccess: () => setSnack({ open: true, message: 'User banned.' }),
                  onError: () => setSnack({ open: true, message: 'Failed to ban user.' }),
                });
              }
              setConfirm({ open: false });
            }}
          >
            Confirm
          </Button>
        </DialogActions>
      </Dialog>
    </TableContainer>
  );
}


