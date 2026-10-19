import { ZonePage } from "@eam/ui";

export default function Page(props: { params: Promise<{ slug?: string[] }> }) {
  return <ZonePage zone="renkon" params={props.params} />;
}
