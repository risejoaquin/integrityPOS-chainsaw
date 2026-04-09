import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Grid, Paper, Typography, Box, Card, CardContent,
  List, ListItem, ListItemText, Divider, Button, Drawer, ListItemButton, ListItemIcon, Text
} from '@mui/material';
import DashboardIcon from '@mui/icons-material/Dashboard';
import ShoppingCartIcon from '@mui/icons-material/ShoppingCart';
import InventoryIcon from '@mui/icons-material/Inventory';
import LogoutIcon from '@mui/icons-material/Logout';
import axios from 'axios';

const drawerWidth = 240;

function Dashboard({ onLogout, backendHealthy }) {
  const navigate = useNavigate();
  const [stats, setStats] = useState({
    totalSales: 0,
    totalRevenue: 0,
    activeSession: null,
    recentSales: []
  });

  useEffect(() => {
    loadDashboardData();
  }, []);

  const loadDashboardData = async () => {
    try {
      const token = localStorage.getItem('token');
      const config = { headers: { Authorization: `Bearer ${token}` } };

      // Load session info
      const sessionResponse = await axios.get('http://localhost:8080/session/current', config);
      setStats(prev => ({ ...prev, activeSession: sessionResponse.data }));

      // Mock stats for now
      setStats(prev => ({
        ...prev,
        totalSales: 42,
        totalRevenue: 125000,
        recentSales: [
          { id: 'sale1', total: 25000, time: '10:30 AM' },
          { id: 'sale2', total: 15000, time: '10:25 AM' },
          { id: 'sale3', total: 35000, time: '10:20 AM' }
        ]
      }));
    } catch (error) {
      console.error('Failed to load dashboard data:', error);
    }
  };

  const handleLogout = () => {
    onLogout();
    navigate('/login');
  };

  const menuItems = [
    { text: 'Dashboard', icon: <DashboardIcon />, path: '/dashboard' },
    { text: 'Ventas', icon: <ShoppingCartIcon />, path: '/sales' },
    { text: 'Inventario', icon: <InventoryIcon />, path: '/inventory' },
  ];

  return (
    <Box sx={{ display: 'flex' }}>
      <Drawer
        sx={{
          width: drawerWidth,
          flexShrink: 0,
          '& .MuiDrawer-paper': {
            width: drawerWidth,
            boxSizing: 'border-box',
          },
        }}
        variant="permanent"
        anchor="left"
      >
        <Box sx={{ p: 2 }}>
          <Typography variant="h6">IntegrityPOS</Typography>
          <Typography variant="body2" color={backendHealthy ? 'success.main' : 'error.main'}>
            Backend: {backendHealthy ? 'Conectado' : 'Desconectado'}
          </Typography>
        </Box>
        <Divider />
        <List>
          {menuItems.map((item) => (
            <ListItem key={item.text} disablePadding>
              <ListItemButton onClick={() => navigate(item.path)}>
                <ListItemIcon>{item.icon}</ListItemIcon>
                <ListItemText primary={item.text} />
              </ListItemButton>
            </ListItem>
          ))}
        </List>
        <Divider />
        <List>
          <ListItem disablePadding>
            <ListItemButton onClick={handleLogout}>
              <ListItemIcon><LogoutIcon /></ListItemIcon>
              <ListItemText primary="Cerrar Sesión" />
            </ListItemButton>
          </ListItem>
        </List>
      </Drawer>
      <Box component="main" sx={{ flexGrow: 1, p: 3 }}>
        <Typography variant="h4" gutterBottom>
          Dashboard
        </Typography>

        <Grid container spacing={3}>
          {/* Stats Cards */}
        <Grid item xs={12} md={4}>
          <Card>
            <CardContent>
              <Typography color="textSecondary" gutterBottom>
                Ventas Hoy
              </Typography>
              <Typography variant="h5">
                {stats.totalSales}
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={4}>
          <Card>
            <CardContent>
              <Typography color="textSecondary" gutterBottom>
                Ingresos Totales
              </Typography>
              <Typography variant="h5">
                ${(stats.totalRevenue / 100).toFixed(2)}
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={4}>
          <Card>
            <CardContent>
              <Typography color="textSecondary" gutterBottom>
                Sesión Activa
              </Typography>
              <Typography variant="h5">
                {stats.activeSession ? '✅' : '❌'}
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        {/* Recent Sales */}
        <Grid item xs={12} md={6}>
          <Paper sx={{ p: 2 }}>
            <Typography variant="h6" gutterBottom>
              Ventas Recientes
            </Typography>
            <List>
              {stats.recentSales.map((sale, index) => (
                <React.Fragment key={sale.id}>
                  <ListItem>
                    <ListItemText
                      primary={`Venta ${sale.id}`}
                      secondary={`$${(sale.total / 100).toFixed(2)} - ${sale.time}`}
                    />
                  </ListItem>
                  {index < stats.recentSales.length - 1 && <Divider />}
                </React.Fragment>
              ))}
            </List>
          </Paper>
        </Grid>

        {/* Quick Actions */}
        <Grid item xs={12} md={6}>
          <Paper sx={{ p: 2 }}>
            <Typography variant="h6" gutterBottom>
              Acciones Rápidas
            </Typography>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
              <Typography variant="body2" color="textSecondary">
                • Abrir nueva sesión de caja
              </Typography>
              <Typography variant="body2" color="textSecondary">
                • Iniciar venta
              </Typography>
              <Typography variant="body2" color="textSecondary">
                • Ver inventario
              </Typography>
              <Typography variant="body2" color="textSecondary">
                • Generar reporte
              </Typography>
            </Box>
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
}

export default Dashboard;