import { createLazyFileRoute } from "@tanstack/react-router";

export const Route = createLazyFileRoute("/")({
	component: Index,
});

function Index() {
	return (
		<div className="">
			<h3 className="text-red-500">Welcome Home!</h3>
		</div>
	);
}
