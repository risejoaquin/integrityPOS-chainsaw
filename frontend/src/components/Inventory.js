import React, { useState, useEffect } from 'react';
import {
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Paper, Typography, Box, Button, TextField, Modal, Grid
} from '@mui/material';
import axios from 'axios';

function Inventory() {
  const [products, setProducts] = useState([]);
  const [search, setSearch] = useState('');
  const [open, setOpen] = useState(false);
  const [newProduct, setNewProduct] = useState({ sku: '', name: '', price: 0, stock: 0 });

  useEffect(() => {
    loadProducts();
  }, []);

  const loadProducts = async () => {
    try {
      const token = localStorage.getItem('token');
      const config = { headers: { Authorization: `Bearer ${token}` } };
      const response = await axios.get('http://localhost:8080/products', config);
      setProducts(response.data);
    } catch (error) {
      console.error('Failed to load products:', error);
      // Mock data
      setProducts([
        { sku: '001', name: 'Producto 1', price: 10000, stock: 50, reserved: 5 },
        { sku: '002', name: 'Producto 2', price: 20000, stock: 30, reserved: 2 },
      ]);
    }
  };

  const handleAddProduct = async () => {
    // Implement add product
    setOpen(false);
  };

  const filteredProducts = products.filter(p =>
    p.name.toLowerCase().includes(search.toLowerCase()) ||
    p.sku.includes(search)
  );

  return (
    <Box>
      <Typography variant="h4" gutterBottom>
        Gestión de Inventario
      </Typography>
      <Box sx={{ mb: 2, display: 'flex', gap: 2 }}>
        <TextField
          label="Buscar"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          variant="outlined"
        />
        <Button variant="contained" onClick={() => setOpen(true)}>
          Añadir Producto
        </Button>
      </Box>
      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>SKU</TableCell>
              <TableCell>Nombre</TableCell>
              <TableCell>Stock Real</TableCell>
              <TableCell>Reservaciones</TableCell>
              <TableCell>Stock Disponible</TableCell>
              <TableCell>Precio</TableCell>
              <TableCell>Acciones</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredProducts.map((product) => (
              <TableRow key={product.sku}>
                <TableCell>{product.sku}</TableCell>
                <TableCell>{product.name}</TableCell>
                <TableCell>{product.stock}</TableCell>
                <TableCell>{product.reserved || 0}</TableCell>
                <TableCell>{product.stock - (product.reserved || 0)}</TableCell>
                <TableCell>${(product.price / 100).toFixed(2)}</TableCell>
                <TableCell>
                  <Button size="small">Editar</Button>
                  <Button size="small" color="error">Eliminar</Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

      <Modal open={open} onClose={() => setOpen(false)}>
        <Box sx={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', width: 400, bgcolor: 'background.paper', p: 4 }}>
          <Typography variant="h6" gutterBottom>Añadir Producto</Typography>
          <Grid container spacing={2}>
            <Grid item xs={12}>
              <TextField fullWidth label="SKU" value={newProduct.sku} onChange={(e) => setNewProduct({...newProduct, sku: e.target.value})} />
            </Grid>
            <Grid item xs={12}>
              <TextField fullWidth label="Nombre" value={newProduct.name} onChange={(e) => setNewProduct({...newProduct, name: e.target.value})} />
            </Grid>
            <Grid item xs={6}>
              <TextField fullWidth label="Precio" type="number" value={newProduct.price} onChange={(e) => setNewProduct({...newProduct, price: parseInt(e.target.value)})} />
            </Grid>
            <Grid item xs={6}>
              <TextField fullWidth label="Stock" type="number" value={newProduct.stock} onChange={(e) => setNewProduct({...newProduct, stock: parseInt(e.target.value)})} />
            </Grid>
            <Grid item xs={12}>
              <Button fullWidth variant="contained" onClick={handleAddProduct}>Guardar</Button>
            </Grid>
          </Grid>
        </Box>
      </Modal>
    </Box>
  );
}

export default Inventory;