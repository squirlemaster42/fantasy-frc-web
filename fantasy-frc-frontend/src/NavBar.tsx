import { Link, useNavigate } from 'react-router-dom';
import { useUser } from './context/UserContext';

const NavBar = () => {
  const { user, logout } = useUser();
  //const location = useLocation();
  const navigate = useNavigate();

  const handleLogout = () => {
    logout();
    navigate('/');
  };

  return (
    <nav
      style={{
        position: 'fixed',        // 🧠 keep it at the top
        top: 0,
        left: 0,
        right: 0,
        height: '60px',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: '0 20px',
        color: 'white',
        zIndex: 1000,
      }}
      className="bg-gray-600"
    >
      <div style={{ display: 'flex', gap: '1rem' }}>
        <Link to="/home" style={{ color: 'white', textDecoration: 'none' }}>Fantasy FRC</Link>
        {user.isLoggedIn && (
          <Link to="/drafts" style={{ color: 'white', textDecoration: 'none' }}>
            Drafts
          </Link>
        )}
      </div>
      <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
        {user.isLoggedIn && (
          <>
            <span>Welcome, {user.name}</span>
                <button
                    onClick={handleLogout}
                    className="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded"
                >
                Logout
            </button>
          </>
        )}
      </div>
    </nav>
  );
};

export default NavBar;
