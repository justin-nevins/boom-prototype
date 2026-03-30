import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { AuthProvider } from './context/AuthContext';
import Home from './pages/Home';
import Room from './pages/Room';
import Login from './pages/Login';
import Join from './pages/Join';
import './index.css';

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/login" element={<Login />} />
          <Route path="/join/:roomName" element={<Join />} />
          <Route path="/room/:roomName" element={<Room />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  );
}
