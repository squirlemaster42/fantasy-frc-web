import './App.css'
import { Route, Routes } from 'react-router-dom'
import HomePage from './home/HomePage';
import LoginPage from './login/LoginPage';
import NotFoundPage from './notfoundpage/NotFoundPage';

function App() {
  return (
    <Routes>
        <Route path="/" element={<LoginPage/>}/>
        <Route path="/home" element={<HomePage/>}/>
        <Route path="*" element={<NotFoundPage/>}/>
    </Routes>
  );
}

export default App
