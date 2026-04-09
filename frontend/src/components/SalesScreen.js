import React, { useState, useEffect } from 'react';
import {
  Grid, Paper, Typography, Box, TextField, Button,
  List, ListItem, ListItemText, IconButton, Card, CardContent,
  Dialog, DialogTitle, DialogContent, DialogActions
} from '@mui/material';
import DeleteIcon from '@mui/icons-material/Delete';
import AddIcon from '@mui/icons-material/Add';
import RemoveIcon from '@mui/icons-material/Remove';
import axios from 'axios';

function SalesScreen() {
  const [cart, setCart] = useState([]);
  const [products, setProducts] = useState([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [paymentDialog, setPaymentDialog] = useState(false);
  const [paymentMethod, setPaymentMethod] = useState('CASH');
  const [paidAmount, setPaidAmount] = useState('');

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
      // Mock products
      setProducts([
        { sku: 'PROD001', name: 'Producto 1', price: 2500, stock: 10 },
        { sku: 'PROD002', name: 'Producto 2', price: 1500, stock: 5 },
        { sku: 'PROD003', name: 'Producto 3', price: 3500, stock: 8 }
      ]);
    }
  };

  const addToCart = (product) => {
    const existing = cart.find(item => item.sku === product.sku);
    if (existing) {
      setCart(cart.map(item =>
        item.sku === product.sku
          ? { ...item, quantity: item.quantity + 1 }
          : item
      ));
    } else {
      setCart([...cart, { ...product, quantity: 1 }]);
    }
  };

  const updateQuantity = (sku, delta) => {
    setCart(cart.map(item =>
      item.sku === sku
        ? { ...item, quantity: Math.max(1, item.quantity + delta) }
        : item
    ));
  };

  const removeFromCart = (sku) => {
    setCart(cart.filter(item => item.sku !== sku));
  };

  const total = cart.reduce((sum, item) => sum + item.price * item.quantity, 0);

  const handlePayment = async () => {
    try {
      const token = localStorage.getItem('token');
      const config = { headers: { Authorization: `Bearer ${token}` } };
      await axios.post('http://localhost:8080/sale', {
        session_id: 'current', // Assume current session
        cajero_id: 'admin',
        terminal_id: 'terminal1',
        payment_method: paymentMethod,
        items: cart.map(item => ({ sku: item.sku, quantity: item.quantity }))
      }, config);
      setCart([]);
      setPaymentDialog(false);
      alert('Venta completada');
    } catch (error) {
      console.error('Payment failed:', error);
      alert('Error en el pago');
    }
  };

  const filteredProducts = products.filter(p =>
    p.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    p.sku.includes(searchTerm)
  );

  return (
    <Box sx={{ display: 'flex', height: '80vh' }}>
      {/* Left side: Products */}
      <Box sx={{ flex: 2, p: 2 }}>
        <Typography variant="h5" gutterBottom>Punto de Venta</Typography>
        <TextField
          fullWidth
          label="Buscar productos"
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          sx={{ mb: 2 }}
        />
        <Grid container spacing={2}>
          {filteredProducts.map((product) => (
            <Grid item xs={12} sm={6} md={4} key={product.sku}>
              <Card onClick={() => addToCart(product)} sx={{ cursor: 'pointer' }}>
                <CardContent>
                  <Typography variant="h6">{product.name}</Typography>
                  <Typography>${(product.price / 100).toFixed(2)}</Typography>
                  <Typography variant="body2">Stock: {product.stock}</Typography>
                </CardContent>
              </Card>
            </Grid>
          ))}
        </Grid>
      </Box>

      {/* Right side: Cart */}
      <Box sx={{ flex: 1, p: 2, borderLeft: 1, borderColor: 'divider' }}>
        <Typography variant="h5" gutterBottom>Ticket</Typography>
        <List>
          {cart.map((item) => (
            <ListItem key={item.sku} secondaryAction={
              <IconButton edge="end" onClick={() => removeFromCart(item.sku)}>
                <DeleteIcon />
              </IconButton>
            }>
              <ListItemText
                primary={item.name}
                secondary={`$${(item.price / 100).toFixed(2)} x ${item.quantity} = $${((item.price * item.quantity) / 100).toFixed(2)}`}
              />
              <Box sx={{ display: 'flex', alignItems: 'center' }}>
                <IconButton onClick={() => updateQuantity(item.sku, -1)}>
                  <RemoveIcon />
                </IconButton>
                <Typography sx={{ mx: 1 }}>{item.quantity}</Typography>
                <IconButton onClick={() => updateQuantity(item.sku, 1)}>
                  <AddIcon />
                </IconButton>
              </Box>
            </ListItem>
          ))}
        </List>
        <Typography variant="h6" sx={{ mt: 2 }}>
          Total: ${(total / 100).toFixed(2)}
        </Typography>
        <Button
          fullWidth
          variant="contained"
          onClick={() => setPaymentDialog(true)}
          disabled={cart.length === 0}
          sx={{ mt: 2 }}
        >
          Pagar en Efectivo
        </Button>
        <Button
          fullWidth
          variant="outlined"
          onClick={() => { setPaymentMethod('CARD'); setPaymentDialog(true); }}
          disabled={cart.length === 0}
          sx={{ mt: 1 }}
        >
          Pagar con Tarjeta
        </Button>
      </Box>

      <Dialog open={paymentDialog} onClose={() => setPaymentDialog(false)}>
        <DialogTitle>Confirmar Pago</DialogTitle>
        <DialogContent>
          <Typography>Total: ${(total / 100).toFixed(2)}</Typography>
          <TextField
            fullWidth
            label="Monto Pagado"
            type="number"
            value={paidAmount}
            onChange={(e) => setPaidAmount(e.target.value)}
            sx={{ mt: 2 }}
          />
          <Typography>Cambio: ${Math.max(0, (parseFloat(paidAmount) || 0) - total / 100).toFixed(2)}</Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setPaymentDialog(false)}>Cancelar</Button>
          <Button onClick={handlePayment} variant="contained">Confirmar</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

export default SalesScreen;

  const updateQuantity = (sku, quantity) => {
    if (quantity <= 0) {
      removeFromCart(sku);
    } else {
      setCart(cart.map(item =>
        item.sku === sku ? { ...item, quantity } : item
      ));
    }
  };

  const getTotal = () => {
    return cart.reduce((sum, item) => sum + (item.price * item.quantity), 0);
  };

  const handlePayment = async () => {
    const total = getTotal();
    const paid = parseFloat(paidAmount) * 100; // to cents

    if (paid < total) {
      alert('Monto insuficiente');
      return;
    }

    try {
      const saleData = {
        session_id: 'session1', // TODO: get from context
        cajero_id: 'cajero1',
        terminal_id: 'terminal1',
        payment_method: paymentMethod,
        items: cart.map(item => ({
          sku: item.sku,
          quantity: item.quantity
        })),
        paid_cents: paid
      };

      const response = await window.electronAPI.apiCall('POST', '/sale', saleData);
      if (response.success) {
        alert('Venta completada!');
        setCart([]);
        setPaymentDialog(false);
        setPaidAmount('');
      } else {
        alert('Error en la venta: ' + response.error);
      }
    } catch (error) {
      alert('Error: ' + error.message);
    }
  };

  const filteredProducts = products.filter(product =>
    product.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
    product.sku.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <Box>
      <Typography variant="h4" gutterBottom>
        Nueva Venta
      </Typography>

      <Grid container spacing={3}>
        {/* Product Search */}
        <Grid item xs={12} md={6}>
          <Paper sx={{ p: 2 }}>
            <Typography variant="h6" gutterBottom>
              Buscar Productos
            </Typography>
            <TextField
              fullWidth
              label="Buscar por nombre o SKU"
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              sx={{ mb: 2 }}
            />
            <List sx={{ maxHeight: 400, overflow: 'auto' }}>
              {filteredProducts.map((product) => (
                <ListItem key={product.sku}>
                  <ListItemText
                    primary={product.name}
                    secondary={`$${ (product.price / 100).toFixed(2) } - Stock: ${product.stock}`}
                  />
                  <IconButton onClick={() => addToCart(product)}>
                    <AddIcon />
                  </IconButton>
                </ListItem>
              ))}
            </List>
          </Paper>
        </Grid>

        {/* Cart */}
        <Grid item xs={12} md={6}>
          <Paper sx={{ p: 2 }}>
            <Typography variant="h6" gutterBottom>
              Carrito
            </Typography>
            <List sx={{ maxHeight: 300, overflow: 'auto' }}>
              {cart.map((item) => (
                <ListItem key={item.sku}>
                  <ListItemText
                    primary={item.name}
                    secondary={`Cant: ${item.quantity} x $${(item.price / 100).toFixed(2)} = $${((item.price * item.quantity) / 100).toFixed(2)}`}
                  />
                  <IconButton onClick={() => removeFromCart(item.sku)}>
                    <DeleteIcon />
                  </IconButton>
                </ListItem>
              ))}
            </List>
            <Box sx={{ mt: 2 }}>
              <Typography variant="h6">
                Total: ${(getTotal() / 100).toFixed(2)}
              </Typography>
              <Button
                variant="contained"
                fullWidth
                sx={{ mt: 1 }}
                onClick={() => setPaymentDialog(true)}
                disabled={cart.length === 0}
              >
                Pagar
              </Button>
            </Box>
          </Paper>
        </Grid>
      </Grid>

      {/* Payment Dialog */}
      <Dialog open={paymentDialog} onClose={() => setPaymentDialog(false)}>
        <DialogTitle>Pago</DialogTitle>
        <DialogContent>
          <Typography gutterBottom>
            Total a pagar: ${(getTotal() / 100).toFixed(2)}
          </Typography>
          <TextField
            fullWidth
            label="Monto pagado"
            type="number"
            value={paidAmount}
            onChange={(e) => setPaidAmount(e.target.value)}
            sx={{ mt: 2 }}
          />
          <Typography sx={{ mt: 1 }}>
            Cambio: ${((parseFloat(paidAmount || 0) * 100 - getTotal()) / 100).toFixed(2)}
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setPaymentDialog(false)}>Cancelar</Button>
          <Button onClick={handlePayment} variant="contained">
            Confirmar Pago
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

export default SalesScreen;