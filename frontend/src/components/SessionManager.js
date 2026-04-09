import React, { useState, useEffect } from 'react';
import {
  Box, Typography, Card, CardContent, Button, TextField,
  Grid, Dialog, DialogTitle, DialogContent, DialogActions,
  List, ListItem, ListItemText, Divider
} from '@mui/material';

function SessionManager() {
  const [currentSession, setCurrentSession] = useState(null);
  const [openDialog, setOpenDialog] = useState(false);
  const [dialogType, setDialogType] = useState(''); // 'open' or 'close'
  const [formData, setFormData] = useState({
    cajero_id: '',
    terminal_id: '',
    initial_cash: '',
    real_cash: ''
  });
  const [movements, setMovements] = useState([]);

  useEffect(() => {
    checkCurrentSession();
  }, []);

  const checkCurrentSession = async () => {
    try {
      const response = await window.electronAPI.apiCall('GET', '/session/current');
      if (response.success) {
        setCurrentSession(response.data);
        loadMovements(response.data.id);
      }
    } catch (error) {
      console.error('Failed to check session:', error);
    }
  };

  const loadMovements = async (sessionId) => {
    try {
      // Mock movements for now
      setMovements([
        { type: 'DEPOSIT', amount: 50000, reason: 'Fondo inicial', time: '09:00 AM' },
        { type: 'WITHDRAWAL', amount: 10000, reason: 'Compra insumos', time: '11:30 AM' }
      ]);
    } catch (error) {
      console.error('Failed to load movements:', error);
    }
  };

  const handleOpenSession = async () => {
    try {
      const data = {
        session_id: `session_${Date.now()}`,
        cajero_id: formData.cajero_id,
        terminal_id: formData.terminal_id,
        initial_cash: parseFloat(formData.initial_cash) * 100, // to cents
        expected_cash: parseFloat(formData.initial_cash) * 100
      };

      const response = await window.electronAPI.apiCall('POST', '/session/open', data);
      if (response.success) {
        setCurrentSession(response.data);
        setOpenDialog(false);
        setFormData({ cajero_id: '', terminal_id: '', initial_cash: '', real_cash: '' });
      } else {
        alert('Error al abrir sesión: ' + response.error);
      }
    } catch (error) {
      alert('Error: ' + error.message);
    }
  };

  const handleCloseSession = async () => {
    try {
      const data = {
        session_id: currentSession.id,
        real_cash: parseFloat(formData.real_cash) * 100
      };

      const response = await window.electronAPI.apiCall('POST', '/session/close', data);
      if (response.success) {
        setCurrentSession(null);
        setMovements([]);
        setOpenDialog(false);
        setFormData({ cajero_id: '', terminal_id: '', initial_cash: '', real_cash: '' });
      } else {
        alert('Error al cerrar sesión: ' + response.error);
      }
    } catch (error) {
      alert('Error: ' + error.message);
    }
  };

  const openDialogFor = (type) => {
    setDialogType(type);
    setOpenDialog(true);
  };

  return (
    <Box>
      <Typography variant="h4" gutterBottom>
        Gestión de Sesión
      </Typography>

      <Grid container spacing={3}>
        {/* Current Session Status */}
        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Estado de Sesión
              </Typography>
              {currentSession ? (
                <Box>
                  <Typography>ID: {currentSession.id}</Typography>
                  <Typography>Cajero: {currentSession.cajero_id}</Typography>
                  <Typography>Terminal: {currentSession.terminal_id}</Typography>
                  <Typography>
                    Efectivo Inicial: ${(currentSession.initial_cash / 100).toFixed(2)}
                  </Typography>
                  <Typography>
                    Estado: {currentSession.closed_at ? 'Cerrada' : 'Abierta'}
                  </Typography>
                  <Button
                    variant="outlined"
                    color="secondary"
                    sx={{ mt: 2 }}
                    onClick={() => openDialogFor('close')}
                    disabled={!!currentSession.closed_at}
                  >
                    Cerrar Sesión
                  </Button>
                </Box>
              ) : (
                <Box>
                  <Typography color="textSecondary">
                    No hay sesión activa
                  </Typography>
                  <Button
                    variant="contained"
                    sx={{ mt: 2 }}
                    onClick={() => openDialogFor('open')}
                  >
                    Abrir Sesión
                  </Button>
                </Box>
              )}
            </CardContent>
          </Card>
        </Grid>

        {/* Movements */}
        <Grid item xs={12} md={6}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Movimientos
              </Typography>
              <List sx={{ maxHeight: 300, overflow: 'auto' }}>
                {movements.map((movement, index) => (
                  <React.Fragment key={index}>
                    <ListItem>
                      <ListItemText
                        primary={`${movement.type === 'DEPOSIT' ? 'Depósito' : 'Retiro'}: $${(movement.amount / 100).toFixed(2)}`}
                        secondary={`${movement.reason} - ${movement.time}`}
                      />
                    </ListItem>
                    {index < movements.length - 1 && <Divider />}
                  </React.Fragment>
                ))}
              </List>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Dialog for Open/Close Session */}
      <Dialog open={openDialog} onClose={() => setOpenDialog(false)}>
        <DialogTitle>
          {dialogType === 'open' ? 'Abrir Sesión' : 'Cerrar Sesión'}
        </DialogTitle>
        <DialogContent>
          {dialogType === 'open' ? (
            <Box sx={{ pt: 1 }}>
              <TextField
                fullWidth
                label="ID Cajero"
                value={formData.cajero_id}
                onChange={(e) => setFormData({...formData, cajero_id: e.target.value})}
                sx={{ mb: 2 }}
              />
              <TextField
                fullWidth
                label="ID Terminal"
                value={formData.terminal_id}
                onChange={(e) => setFormData({...formData, terminal_id: e.target.value})}
                sx={{ mb: 2 }}
              />
              <TextField
                fullWidth
                label="Efectivo Inicial"
                type="number"
                value={formData.initial_cash}
                onChange={(e) => setFormData({...formData, initial_cash: e.target.value})}
              />
            </Box>
          ) : (
            <Box sx={{ pt: 1 }}>
              <Typography gutterBottom>
                Efectivo esperado: ${(currentSession?.expected_cash / 100).toFixed(2)}
              </Typography>
              <TextField
                fullWidth
                label="Efectivo Real"
                type="number"
                value={formData.real_cash}
                onChange={(e) => setFormData({...formData, real_cash: e.target.value})}
              />
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpenDialog(false)}>Cancelar</Button>
          <Button
            onClick={dialogType === 'open' ? handleOpenSession : handleCloseSession}
            variant="contained"
          >
            {dialogType === 'open' ? 'Abrir' : 'Cerrar'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

export default SessionManager;