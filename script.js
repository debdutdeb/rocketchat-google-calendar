class Script {
  getDuration(startTime, endTime) {
    return (endTime - startTime) / 60000;
  }
  process_incoming_request({ request }) {
    const maxAttendees = 5;
    const event = request.content;
    let msg = "### Upcoming Event\n";

    let startTime = Date.parse(event.start.dateTime);
    let endTime = Date.parse(event.end.dateTime);

    msg +=
      `*Summary:* ${event.summary}\n` +
      `*Start Time:* ${Date(startTime).toString()}\n` +
      `*End Time:* ${Date(endTime).toString()}\n` +
      `*Duration:* ${this.getDuration(startTime, endTime)} minutes\n` +
      `*Due In:* ${Math.floor(
        this.getDuration(+new Date(), startTime)
      )} minutes\n` +
      `${
        event.attendees
          ? event.attendees.length > maxAttendees
            ? `*Attendees:* ${event.attendees.length} attendees\n`
            : `*Attendees:* ${event.attendees
                .map((attendee) =>
                  attendee.displayName ? attendee.displayName : attendee.email
                )
                .join(", ")}\n`
          : ""
      }` +
      `*Calendar Link:* ${event.htmlLink}\n` +
      `${event.hangoutLink ? `*Meet Link:* ${event.hangoutLink}\n` : ""}` +
      `${event.location ? `*Location:* ${event.location}\n` : ""}` +
      `${event.description ? `*Description:* ${event.description}\n` : ""}` +
      `\n`;

    return {
      content: {
        text: msg,
      },
    };
  }
}
