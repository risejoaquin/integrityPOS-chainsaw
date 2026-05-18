import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { Toaster } from 'sonner'
import App from './App'
import './index.css'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
    <Toaster
      position="top-right"
      richColors
      closeButton
      toastOptions={{
        duration: 3000,
        style: {
          background: '#18181b',
          border: '1px solid #3f3f46',
          color: '#f4f4f5',
        },
      }}
    />
  </StrictMode>,
)