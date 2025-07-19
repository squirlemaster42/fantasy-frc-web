import { useUser } from '../context/UserContext';

const HomePage = () => {
    const { loginAs } = useUser()

    return (
        <div>
            Welcome to Fantasy FRC
            <button onClick={() => loginAs('Alice')}>Login as Alice</button>
        </div>
    );
}

export default HomePage
