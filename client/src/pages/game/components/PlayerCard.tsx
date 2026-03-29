import type { GameState, PlayerState } from "../../../models/game-state"

export interface PlayerCardProps {
    playerState: PlayerState;
    gameState: GameState;
}

export default function PlayerCard({ playerState, gameState }: PlayerCardProps) {
    const { id, cardCount, isAlive, isOnline } = playerState;

    const isMe = gameState.playerId === id;
    const isTurn = gameState.turnId === id;

    let avatarStyles = "w-20 h-20 rounded-full flex items-center justify-center text-2xl font-bold shadow-md relative transition-all duration-300 ";

    if (!isAlive) {
        avatarStyles += "bg-gray-200 text-gray-400 grayscale";
    } else {
        avatarStyles += "bg-indigo-500 text-white shadow-lg";
    }

    if (isTurn && isAlive && gameState.inProgress) {
        avatarStyles += " ring-4 ring-offset-4 ring-yellow-400";
    }

    return (
        <div className={`flex flex-col items-center gap-2 ${!isAlive ? 'opacity-70' : ''}`}>
            <div className="relative mt-2">
                <div className={avatarStyles}>
                    P{id}

                    {!isAlive && (
                        <span className="absolute text-red-500/80 text-6xl font-light rotate-12 drop-shadow-sm select-none">
                            ✗
                        </span>
                    )}
                </div>

                <div
                    className={`absolute bottom-0 right-0 w-5 h-5 rounded-full border-2 border-white ${isOnline ? 'bg-green-500' : 'bg-gray-400'
                        }`}
                    title={isOnline ? "Online" : "Offline"}
                />

                {isAlive && gameState.inProgress && (
                    <div
                        className="absolute -top-2 -right-2 bg-slate-800 text-white text-xs font-bold w-7 h-7 rounded-full flex items-center justify-center border-2 border-white shadow-sm"
                        title="Cards in hand"
                    >
                        {cardCount}
                    </div>
                )}
            </div>

            <div className="text-center flex flex-col items-center mt-1">

                <div className="flex items-center gap-2">
                    <span className={`font-bold ${isAlive ? 'text-gray-800' : 'text-gray-500 line-through'}`}>
                        Player {id}
                    </span>
                    {isMe && (
                        <span className="text-[10px] font-black uppercase bg-indigo-100 text-indigo-700 px-2 py-0.5 rounded-full tracking-wider">
                            You
                        </span>
                    )}
                </div>

                <div className="h-4 mt-1">
                    {isTurn && isAlive && gameState.inProgress && (
                        <span className="text-[11px] font-bold text-yellow-600 uppercase tracking-widest flex items-center gap-1">
                            <span className="w-1.5 h-1.5 rounded-full bg-yellow-500 animate-pulse"></span>
                            Current Turn
                        </span>
                    )}
                </div>
            </div>

        </div>
    );
}